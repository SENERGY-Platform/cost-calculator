/*
 *    Copyright 2023 InfAI (CC SES)
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package controller

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
)

const allocationOverviewMonthKey = "allocation_overview_month"
const allocationContainerMonthKey = "allocation_container_month"
const allocationControllerMonthKey = "allocation_controller_month"

func init() {
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace",
		})
		if err != nil {
			log.Println("WARNING: Could not prefetch cost overview: " + err.Error())
			return err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			log.Println("WARNING: Could not prefetch cost overview, invalid allocation response: " + err.Error())
			return err
		}
		c.cacheMux.Lock()
		defer c.cacheMux.Unlock()
		c.cache[allocationOverviewMonthKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace,controller",
		})
		if err != nil {
			log.Println("WARNING: Could not prefetch cost pods: " + err.Error())
			return err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			log.Println("WARNING: Could not prefetch cost pods, invalid allocation response: " + err.Error())
			return err
		}
		c.cacheMux.Lock()
		defer c.cacheMux.Unlock()
		c.cache[allocationControllerMonthKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace,controller,container",
		})
		if err != nil {
			log.Println("WARNING: Could not prefetch cost containers: " + err.Error())
			return err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			log.Println("WARNING: Could not prefetch cost containers, invalid allocation response: " + err.Error())
			return err
		}
		c.cacheMux.Lock()
		defer c.cacheMux.Unlock()
		c.cache[allocationContainerMonthKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
}

func (c *Controller) GetCostOverview(userid string) (res model.CostOverview, err error) {
	c.cacheMux.Lock()
	cached, ok := c.cache[allocationOverviewMonthKey]
	c.cacheMux.Unlock()
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace",
		})
		if err != nil {
			return nil, err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			return nil, err
		}
	}

	l24hEntries, err := c.getCostOverview24h(userid)
	if err != nil {
		return nil, err
	}

	res = model.CostOverview{}
	for key, allo := range allocation.Data[0] {
		if key == userid+"/"+c.config.NamespaceAnalytics {
			month := model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
			l24hEntry, ok := l24hEntries[model.CostTypeAnalytics]
			if !ok {
				l24hEntry = model.CostEntry{}
			}
			estimationMonth := predict(month, l24hEntry)
			res[model.CostTypeAnalytics] = model.CostWithEstimation{
				Month:           month,
				EstimationMonth: estimationMonth,
			}
		}
	}
	return res, nil
}

func (c *Controller) GetCostContainers(userid string, costType model.CostType, controllerName string) (res model.CostContainers, err error) {
	var prefix string
	switch costType {
	case model.CostTypeAnalytics:
		prefix = userid + "/" + c.config.NamespaceAnalytics + "/" + controllerName + "/"
	default:
		return nil, errors.New("unknown costType")
	}
	c.cacheMux.Lock()
	cached, ok := c.cache[allocationContainerMonthKey]
	c.cacheMux.Unlock()
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace,controller,container",
		})
		if err != nil {
			return nil, err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			return nil, err
		}
	}
	l24hEntries, err := c.getCostContainers24h(userid, costType, controllerName)
	if err != nil {
		return nil, err
	}

	res = model.CostContainers{}
	for key, allo := range allocation.Data[0] {
		if strings.HasPrefix(key, prefix) {
			month := model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
			l24hEntry, ok := l24hEntries[strings.TrimPrefix(key, prefix)]
			if !ok {
				l24hEntry = model.CostEntry{}
			}
			estimationMonth := predict(month, l24hEntry)
			res[strings.TrimPrefix(key, prefix)] = model.CostWithEstimation{
				Month:           month,
				EstimationMonth: estimationMonth,
			}
		}
	}
	return res, nil
}

func (c *Controller) GetCostControllers(userid string, costType model.CostType) (res model.CostControllers, err error) {
	var prefix string
	switch costType {
	case model.CostTypeAnalytics:
		prefix = userid + "/" + c.config.NamespaceAnalytics + "/"
	default:
		return nil, errors.New("unknown costType")
	}
	c.cacheMux.Lock()
	cached, ok := c.cache[allocationControllerMonthKey]
	c.cacheMux.Unlock()
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "month",
			Aggregate: "label:user,namespace,controller",
		})
		if err != nil {
			return nil, err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			return nil, err
		}
	}
	l24hEntries, err := c.getCostControllers24h(userid, costType)
	if err != nil {
		return nil, err
	}

	res = model.CostControllers{}
	for key, allo := range allocation.Data[0] {
		if strings.HasPrefix(key, prefix) {
			month := model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
			l24hEntry, ok := l24hEntries[strings.TrimPrefix(key, prefix)]
			if !ok {
				l24hEntry = model.CostEntry{}
			}
			estimationMonth := predict(month, l24hEntry)
			res[strings.TrimPrefix(key, prefix)] = model.CostWithEstimation{
				Month:           month,
				EstimationMonth: estimationMonth,
			}
		}
	}
	return res, nil
}

func (c *Controller) GetCostTree(userid string) (res model.CostTree, err error) {
	overview, err := c.GetCostOverview(userid)
	if err != nil {
		return
	}
	res = model.CostTree{}
	for costType, value := range overview {
		controllers, err := c.GetCostControllers(userid, costType)
		if err != nil {
			return res, err
		}
		controllerTree := model.CostTree{}
		for controllerName, controllerCost := range controllers {
			containers, err := c.GetCostContainers(userid, costType, controllerName)
			if err != nil {
				return res, err
			}
			containerTree := model.CostTree{}
			for containerName, containerCost := range containers {
				containerTree[containerName] = model.CostWithChildren{
					CostWithEstimation: containerCost,
				}
			}
			controllerTree[controllerName] = model.CostWithChildren{
				CostWithEstimation: controllerCost,
				Children:           containerTree,
			}
		}
		res[costType] = model.CostWithChildren{
			CostWithEstimation: value,
			Children:           controllerTree,
		}
	}
	return res, nil
}
