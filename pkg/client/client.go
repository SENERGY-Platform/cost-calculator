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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

type Client interface {
	GetTree(token string, skipEstimation bool, start *time.Time, end *time.Time, forUser *string) (model.CostTree, error)
	GetSubTree(token string, costType model.CostType, skipEstimation bool, start *time.Time, end *time.Time, forUser *string) (model.CostTree, error)
	GetFlowEstimation(token string, flowId string) (model.Estimation, error)
	GetFlowEstimations(token string, flowIds []string) ([]model.Estimation, error)
	GetImportEstimation(token string, importTypeId string) (model.Estimation, error)
	GetImportEstimations(token string, importTypeIds []string) ([]model.Estimation, error)
}

type impl struct {
	baseUrl string
}

func New(baseUrl string) Client {
	return &impl{baseUrl: baseUrl}
}

func do[T any](req *http.Request) (result T, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, fmt.Errorf("unexpected statuscode %v", resp.StatusCode)
	}
	if resp.ContentLength == 0 {
		return result, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err
	}
	return result, nil
}
