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

package model

type CostWithEstimation struct {
	Month           CostEntry `json:"month"`
	EstimationMonth CostEntry `json:"estimation_month"`
}

type CostEntry struct {
	Cpu      float64 `json:"cpu,omitempty"`
	Ram      float64 `json:"ram,omitempty"`
	Storage  float64 `json:"storage,omitempty"`
	Requests float64 `json:"requests,omitempty"`
}

func (a *CostEntry) Add(b CostEntry) {
	a.Cpu += b.Cpu
	a.Ram += b.Ram
	a.Storage += b.Storage
	a.Requests += b.Requests
}

type CostOverview = map[CostType]CostWithEstimation

type CostOverviewEntries = map[CostType]CostEntry

type CostType = string

const CostTypeAnalytics CostType = "analytics"
const CostTypeImports CostType = "imports"
const CostTypeApiCalls CostType = "API Calls"
const CostTypeExports CostType = "Exports"
const CostTypeDevices CostType = "Devices"
const CostTypeProcesses CostType = "process"

type CostControllers = map[string]CostWithEstimation

type CostControllerEntries = map[string]CostEntry

type CostWithChildren struct {
	CostWithEstimation
	Children map[string]CostWithChildren `json:"children,omitempty"`
}

type CostTree map[string]CostWithChildren

func (a *CostWithEstimation) Add(b CostWithEstimation) {
	a.Month.Add(b.Month)
	a.EstimationMonth.Add(b.EstimationMonth)
}
