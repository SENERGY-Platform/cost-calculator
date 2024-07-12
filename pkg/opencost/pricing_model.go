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
	"encoding/json"
	"os"
	"strconv"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
)

type PricingModel struct {
	CPU                   string `json:"CPU"`
	GPU                   string `json:"GPU"`
	RAM                   string `json:"RAM"`
	Description           string `json:"description"`
	InternetNetworkEgress string `json:"internetNetworkEgress"`
	RegionNetworkEgress   string `json:"regionNetworkEgress"`
	SpotCPU               string `json:"spotCPU"`
	SpotRAM               string `json:"spotRAM"`
	Storage               string `json:"storage"`
	ZoneNetworkEgress     string `json:"zoneNetworkEgress"`
	Provider              string `json:"provider"`
}

func (m *PricingModel) toModel() (res model.PricingModel, err error) {
	res = model.PricingModel{
		Description: m.Description,
		Provider:    m.Provider,
	}
	res.CPU, err = strconv.ParseFloat(m.CPU, 64)
	if err != nil {
		return
	}
	res.GPU, err = strconv.ParseFloat(m.GPU, 64)
	if err != nil {
		return
	}
	res.RAM, err = strconv.ParseFloat(m.RAM, 64)
	if err != nil {
		return
	}
	res.InternetNetworkEgress, err = strconv.ParseFloat(m.InternetNetworkEgress, 64)
	if err != nil {
		return
	}
	res.RegionNetworkEgress, err = strconv.ParseFloat(m.RegionNetworkEgress, 64)
	if err != nil {
		return
	}
	res.SpotCPU, err = strconv.ParseFloat(m.SpotCPU, 64)
	if err != nil {
		return
	}
	res.SpotRAM, err = strconv.ParseFloat(m.SpotRAM, 64)
	if err != nil {
		return
	}
	res.Storage, err = strconv.ParseFloat(m.Storage, 64)
	if err != nil {
		return
	}
	res.ZoneNetworkEgress, err = strconv.ParseFloat(m.ZoneNetworkEgress, 64)
	if err != nil {
		return
	}
	return
}

func (c *Client) GetPricingModel() (model model.PricingModel, err error) {
	if c.pricingModel != nil {
		return *c.pricingModel, nil
	}
	f, err := os.ReadFile(c.config.PricingModelFilePath)
	if err != nil {
		return model, err
	}
	internalModel := PricingModel{}
	err = json.Unmarshal(f, &internalModel)
	if err != nil {
		return model, err
	}
	model, err = internalModel.toModel()
	if err != nil {
		return model, err
	}
	c.pricingModel = &model
	return
}
