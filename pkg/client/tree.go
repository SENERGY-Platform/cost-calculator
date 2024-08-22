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
	"net/http"
	"strconv"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

func (c *impl) GetTree(token string, skipEstimation bool, start *time.Time, end *time.Time, forUser *string) (model.CostTree, error) {
	url := c.baseUrl + "/tree?skip_estimation=" + strconv.FormatBool(skipEstimation)
	if start != nil {
		url += "&start=" + start.Format(time.RFC3339)
	}
	if end != nil {
		url += "&end=" + end.Format(time.RFC3339)
	}
	if forUser != nil {
		url += "&for_user=" + *forUser
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return nil, err
	}
	return do[model.CostTree](req)
}

func (c *impl) GetSubTree(token string, costType model.CostType, skipEstimation bool, start *time.Time, end *time.Time, forUser *string) (model.CostTree, error) {
	url := c.baseUrl + "/tree/" + costType + "?skip_estimation=" + strconv.FormatBool(skipEstimation)
	if start != nil {
		url += "&start=" + start.Format(time.RFC3339)
	}
	if end != nil {
		url += "&end=" + end.Format(time.RFC3339)
	}
	if forUser != nil {
		url += "&for_user=" + *forUser
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return nil, err
	}
	return do[model.CostTree](req)
}
