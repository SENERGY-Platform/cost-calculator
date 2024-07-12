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

type PricingModel struct {
	CPU                   float64 `json:"CPU"`
	GPU                   float64 `json:"GPU"`
	RAM                   float64 `json:"RAM"`
	Description           string  `json:"description"`
	InternetNetworkEgress float64 `json:"internetNetworkEgress"`
	RegionNetworkEgress   float64 `json:"regionNetworkEgress"`
	SpotCPU               float64 `json:"spotCPU"`
	SpotRAM               float64 `json:"spotRAM"`
	Storage               float64 `json:"storage"`
	ZoneNetworkEgress     float64 `json:"zoneNetworkEgress"`
	Provider              string  `json:"provider"`
}
