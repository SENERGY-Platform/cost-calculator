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

package opencost

import (
	"time"
)

type AllocationOptions struct {
	Window    string
	Aggregate string
}

func (o *AllocationOptions) toQuery() string {
	if o == nil {
		return ""
	}
	return "?window=" + o.Window + "&aggregate=" + o.Aggregate
}

type AllocationPvEntry struct {
	ByteHours  float64 `json:"byteHours"`
	Cost       float64 `json:"cost"`
	ProviderID string  `json:"providerID"`
}

type AllocationEntry struct {
	Name       string `json:"name"`
	Properties struct {
	} `json:"properties"`
	Window struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"window"`
	Start                          time.Time                    `json:"start"`
	End                            time.Time                    `json:"end"`
	Minutes                        float64                      `json:"minutes"`
	CpuCores                       float64                      `json:"cpuCores"`
	CpuCoreRequestAverage          float64                      `json:"cpuCoreRequestAverage"`
	CpuCoreUsageAverage            float64                      `json:"cpuCoreUsageAverage"`
	CpuCoreHours                   float64                      `json:"cpuCoreHours"`
	CpuCost                        float64                      `json:"cpuCost"`
	CpuCostAdjustment              float64                      `json:"cpuCostAdjustment"`
	CpuEfficiency                  float64                      `json:"cpuEfficiency"`
	GpuCount                       float64                      `json:"gpuCount"`
	GpuHours                       float64                      `json:"gpuHours"`
	GpuCost                        float64                      `json:"gpuCost"`
	GpuCostAdjustment              float64                      `json:"gpuCostAdjustment"`
	NetworkTransferBytes           float64                      `json:"networkTransferBytes"`
	NetworkReceiveBytes            float64                      `json:"networkReceiveBytes"`
	NetworkCost                    float64                      `json:"networkCost"`
	NetworkCrossZoneCost           float64                      `json:"networkCrossZoneCost"`
	NetworkCrossRegionCost         float64                      `json:"networkCrossRegionCost"`
	NetworkInternetCost            float64                      `json:"networkInternetCost"`
	NetworkCostAdjustment          float64                      `json:"networkCostAdjustment"`
	LoadBalancerCost               float64                      `json:"loadBalancerCost"`
	LoadBalancerCostAdjustment     float64                      `json:"loadBalancerCostAdjustment"`
	PvBytes                        float64                      `json:"pvBytes"`
	PvByteHours                    float64                      `json:"pvByteHours"`
	PvCost                         float64                      `json:"pvCost"`
	Pvs                            map[string]AllocationPvEntry `json:"pvs"`
	PvCostAdjustment               float64                      `json:"pvCostAdjustment"`
	RamBytes                       float64                      `json:"ramBytes"`
	RamByteRequestAverage          float64                      `json:"ramByteRequestAverage"`
	RamByteUsageAverage            float64                      `json:"ramByteUsageAverage"`
	RamByteHours                   float64                      `json:"ramByteHours"`
	RamCost                        float64                      `json:"ramCost"`
	RamCostAdjustment              float64                      `json:"ramCostAdjustment"`
	RamEfficiency                  float64                      `json:"ramEfficiency"`
	ExternalCost                   float64                      `json:"externalCost"`
	SharedCost                     float64                      `json:"sharedCost"`
	TotalCost                      float64                      `json:"totalCost"`
	TotalEfficiency                float64                      `json:"totalEfficiency"`
	ProportionalAssetResourceCosts struct {
	} `json:"proportionalAssetResourceCosts"`
	LbAllocations       interface{} `json:"lbAllocations"`
	SharedCostBreakdown struct {
	} `json:"sharedCostBreakdown"`
}

type AllocationResponse struct {
	Code   int                          `json:"code"`
	Status string                       `json:"status"`
	Data   []map[string]AllocationEntry `json:"data"`
}
