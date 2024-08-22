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
	"sync"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

func (c *Controller) GetCostControllers(userid string, token string, admin bool, costType model.CostType, skipEstimation bool, start *time.Time, end *time.Time) (res model.CostWithChildren, err error) {
	switch costType {
	case model.CostTypeAnalytics:
		return c.GetAnalyticsTree(userid, skipEstimation, start, end)
	case model.CostTypeImports:
		return c.GetImportsTree(userid, skipEstimation, start, end)
	case model.CostTypeProcesses:
		return c.GetProcessTree(userid, skipEstimation, start, end)
	case model.CostTypeApiCalls:
		return c.GetApiCallsTree(userid, skipEstimation, start, end)
	case model.CostTypeDevices:
		return c.GetDevicesTree(userid, token, skipEstimation, start, end)
	case model.CostTypeExports:
		return c.GetExportsTree(userid, token, admin, skipEstimation, start, end)
	default:
		return res, errors.New("unknown costType")
	}

}

func (c *Controller) GetCostTree(userid string, token string, admin bool, skipEstimation bool, start *time.Time, end *time.Time) (res model.CostTree, err error) {
	res = model.CostTree{}
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	var superErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		analyticsTree, err := c.GetAnalyticsTree(userid, skipEstimation, start, end)
		if err != nil {
			superErr = err
			return
		}
		if analyticsTree.Month.Cpu != 0 || analyticsTree.Month.Ram != 0 || analyticsTree.Month.Storage != 0 {
			res[model.CostTypeAnalytics] = analyticsTree
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		importsTree, err := c.GetImportsTree(userid, skipEstimation, start, end)
		if err != nil {
			superErr = err
			return
		}
		if importsTree.Month.Cpu != 0 || importsTree.Month.Ram != 0 || importsTree.Month.Storage != 0 {
			res[model.CostTypeImports] = importsTree
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		processTree, err := c.GetProcessTree(userid, skipEstimation, start, end)
		if err != nil {
			superErr = err
			return
		}
		if processTree.Month.Cpu != 0 || processTree.Month.Ram != 0 || processTree.Month.Storage != 0 {
			mux.Lock()
			res[model.CostTypeProcesses] = processTree
			mux.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		username, err := c.getUsername(userid)
		if err != nil {
			return
		}
		apiCallsTree, err := c.GetApiCallsTree(username, skipEstimation, start, end)
		if err != nil {
			superErr = err
			return
		}
		if apiCallsTree.Month.Requests != 0 {
			res[model.CostTypeApiCalls] = apiCallsTree
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		devicesTree, err := c.GetDevicesTree(userid, token, skipEstimation, start, end)
		if err != nil {
			superErr = err
			return
		}
		if devicesTree.Month.Cpu != 0 || devicesTree.Month.Ram != 0 || devicesTree.Month.Storage != 0 || devicesTree.Month.Requests != 0 {
			mux.Lock()
			res["Devices"] = devicesTree
			mux.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		exportsTree, err := c.GetExportsTree(userid, token, admin, skipEstimation, start, end)
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
