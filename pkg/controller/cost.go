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
	"strings"
	"sync"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
)

func init() {
	prefetchFn = append(prefetchFn, GetPrefetchAllocationFunction(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace",
	}))
	prefetchFn = append(prefetchFn, GetPrefetchAllocationFunction(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace,controller",
	}))
	prefetchFn = append(prefetchFn, GetPrefetchAllocationFunction(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace,controller,container",
	}))
}

func (c *Controller) GetCostOverview(userid string) (res model.CostOverview, err error) {
	allocation, err := c.GetCachedAllocation(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace",
	})
	if err != nil {
		return nil, err
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
		} else if key == userid+"/"+c.config.NamespaceImports {
			month := model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
			l24hEntry, ok := l24hEntries[model.CostTypeImports]
			if !ok {
				l24hEntry = model.CostEntry{}
			}
			estimationMonth := predict(month, l24hEntry)
			res[model.CostTypeImports] = model.CostWithEstimation{
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
	case model.CostTypeImports:
		prefix = userid + "/" + c.config.NamespaceImports + "/" + controllerName + "/"
	default:
		return nil, errors.New("unknown costType")
	}

	allocation, err := c.GetCachedAllocation(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace,controller,container",
	})
	if err != nil {
		return nil, err
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
	case model.CostTypeImports:
		prefix = userid + "/" + c.config.NamespaceImports + "/"
	default:
		return nil, errors.New("unknown costType")
	}

	return c.GetCostControllersWithFilter(func(key string, allo opencost.AllocationEntry) (use bool, newName string) {
		return strings.HasPrefix(key, prefix), strings.TrimPrefix(key, prefix)
	})
}

type FilterFuncType = func(key string, allo opencost.AllocationEntry) (use bool, newName string)

func (c *Controller) GetCostControllersWithFilter(filter FilterFuncType) (res model.CostControllers, err error) {
	allocation, err := c.GetCachedAllocation(opencost.AllocationOptions{
		Window:    "month",
		Aggregate: "label:user,namespace,controller",
	})
	if err != nil {
		return nil, err
	}

	l24hEntries, err := c.getCostControllers24hWithFilter(filter)
	if err != nil {
		return nil, err
	}

	res = model.CostControllers{}
	for key, allo := range allocation.Data[0] {
		if use, newName := filter(key, allo); use {
			month := model.CostEntry{
				Cpu:     allo.CpuCost,
				Ram:     allo.RamCost,
				Storage: allo.PvCost,
			}
			l24hEntry, ok := l24hEntries[newName]
			if !ok {
				l24hEntry = model.CostEntry{}
			}
			estimationMonth := predict(month, l24hEntry)
			res[newName] = model.CostWithEstimation{
				Month:           month,
				EstimationMonth: estimationMonth,
			}
		}
	}
	return res, nil
}

func (c *Controller) GetCostTree(userid string, token string, admin bool) (res model.CostTree, err error) {
	res = model.CostTree{}
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	var superErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		overview, err := c.GetCostOverview(userid)
		if err != nil {
			superErr = err
			return
		}
		for costType, value := range overview {
			wg.Add(1)
			go func() {
				defer wg.Done()
				controllers, err := c.GetCostControllers(userid, costType)
				if err != nil {
					superErr = err
					return
				}
				controllerTree := model.CostTree{}
				for controllerName, controllerCost := range controllers {
					containers, err := c.GetCostContainers(userid, costType, controllerName)
					if err != nil {
						superErr = err
						return
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
				mux.Lock()
				res[costType] = model.CostWithChildren{
					CostWithEstimation: value,
					Children:           controllerTree,
				}
				mux.Unlock()
			}()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		processCost, err := c.GetProcessTree(userid)
		if err != nil {
			superErr = err
			return
		}
		for key, value := range processCost {
			if value.Month.Cpu != 0 || value.Month.Ram != 0 || value.Month.Storage != 0 {
				mux.Lock()
				res[key] = value
				mux.Unlock()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		username, err := c.getUsername(userid)
		if err != nil {
			return
		}
		apiCallsTree, err := c.GetApiCallsTree(username)
		if err != nil {
			superErr = err
			return
		}
		if apiCallsTree.Month.Requests != 0 {
			res["API Calls"] = apiCallsTree
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		devicesTree, err := c.GetDevicesTree(userid, token)
		if err != nil {
			return
		}
		if devicesTree.Month.Cpu != 0 || devicesTree.Month.Ram != 0 || devicesTree.Month.Storage != 0 {
			mux.Lock()
			res["Devices"] = devicesTree
			mux.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		exportsTree, err := c.GetExportsTree(userid, token, admin)
		if err != nil {
			superErr = err
			return
		}
		if exportsTree.Month.Cpu != 0 || exportsTree.Month.Ram != 0 || exportsTree.Month.Storage != 0 {
			mux.Lock()
			res["Exports"] = exportsTree
			mux.Unlock()
		}
	}()

	wg.Wait()
	return res, superErr
}
