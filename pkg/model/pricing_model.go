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

import (
	"encoding/json"
	"os"
	"strconv"
)

type PricingModel struct {
	CPU     float64 `json:"CPU"`
	RAM     float64 `json:"RAM"`
	Storage float64 `json:"storage"`
}

type pricingModelStr struct {
	CPU         string `json:"CPU"`
	RAM         string `json:"RAM"`
	Description string `json:"description"`
	Storage     string `json:"storage"`
}

func (m *pricingModelStr) toModel() (res PricingModel, err error) {
	res = PricingModel{}
	res.CPU, err = strconv.ParseFloat(m.CPU, 64)
	if err != nil {
		return
	}

	res.RAM, err = strconv.ParseFloat(m.RAM, 64)
	if err != nil {
		return
	}

	res.Storage, err = strconv.ParseFloat(m.Storage, 64)
	if err != nil {
		return
	}

	return
}

func GetPricingModel(filePath string) (model PricingModel, err error) {
	f, err := os.ReadFile(filePath)
	if err != nil {
		return model, err
	}
	internalModel := pricingModelStr{}
	err = json.Unmarshal(f, &internalModel)
	if err != nil {
		return model, err
	}
	model, err = internalModel.toModel()
	if err != nil {
		return model, err
	}
	return
}
