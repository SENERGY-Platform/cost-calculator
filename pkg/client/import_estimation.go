/*
 *    Copyright 2024 InfAI (CC SES)
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

package client

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

func (c *impl) GetImportEstimation(token string, importTypeId string) (model.Estimation, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseUrl+"/estimation/import/"+importTypeId, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return model.Estimation{}, err
	}
	return do[model.Estimation](req)
}
func (c *impl) GetImportEstimations(token string, importTypeIds []string) ([]model.Estimation, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(importTypeIds)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseUrl+"/estimation/import", buf)
	req.Header.Set("Authorization", token)
	if err != nil {
		return nil, err
	}
	return do[[]model.Estimation](req)
}
