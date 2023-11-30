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

const allocationOverview24hKey = "allocation_overview_24h"
const allocationContainer24hKey = "allocation_container_24h"
const allocationController24hKey = "allocation_controller_24h"

func init() {
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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
		c.cache[allocationOverview24hKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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
		c.cache[allocationController24hKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
	prefetchFn = append(prefetchFn, func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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
		c.cache[allocationContainer24hKey] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	})
}

func (c *Controller) getCostOverview24h(userid string) (res model.CostOverviewEntries, err error) {
	cached, ok := c.cache[allocationOverview24hKey]
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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

	res = model.CostOverviewEntries{}
	for key, allo := range allocation.Data[0] {
		if key == userid+"/"+c.config.NamespaceAnalytics {
			res[model.CostTypeAnalytics] = model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
		}
	}
	return res, nil
}

func (c *Controller) getCostContainers24h(userid string, costType model.CostType, controllerName string) (res model.CostContainerEntries, err error) {
	var prefix string
	switch costType {
	case model.CostTypeAnalytics:
		prefix = userid + "/" + c.config.NamespaceAnalytics + "/" + controllerName + "/"
	default:
		return nil, errors.New("unknown costType")
	}
	cached, ok := c.cache[allocationContainer24hKey]
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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
	res = model.CostContainerEntries{}
	for key, allo := range allocation.Data[0] {
		if strings.HasPrefix(key, prefix) {
			res[strings.TrimPrefix(key, prefix)] = model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
		}
	}
	return res, nil
}

func (c *Controller) getCostControllers24h(userid string, costType model.CostType) (res model.CostControllerEntries, err error) {
	var prefix string
	switch costType {
	case model.CostTypeAnalytics:
		prefix = userid + "/" + c.config.NamespaceAnalytics + "/"
	default:
		return nil, errors.New("unknown costType")
	}
	cached, ok := c.cache[allocationController24hKey]
	var allocation opencost.AllocationResponse
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if c.config.Prefetch {
		return nil, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    "24h",
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
	res = model.CostControllerEntries{}
	for key, allo := range allocation.Data[0] {
		if strings.HasPrefix(key, prefix) {
			res[strings.TrimPrefix(key, prefix)] = model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
		}
	}
	return res, nil
}
