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

package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	permissions "github.com/SENERGY-Platform/permission-search/lib/client"
	timescale_wrapper "github.com/SENERGY-Platform/timescale-wrapper/pkg/client"
)

var errUnexpectedReponseFormat = errors.New("unexpected response format")

/*	Limitations:
	- Device cost only considers storage cost.
	- Storage cost is calculated as the storage used at the end of the
	  month as if this was the storage used during the whole month.
	  Any changes in usage over the month will be disregarded.
*/

func (c *Controller) GetDevicesTree(userId string, token string) (result model.CostWithChildren, err error) {
	result = model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{},
		Children:           map[string]model.CostWithChildren{},
	}

	pricingModel, err := c.opencost.GetPricingModel()
	if err != nil {
		return result, err
	}

	hoursInMonthTotal, hoursInMonthProgressed, timeInMonthRemaining := prepExtrapolate()

	limit := 0
	found := 0
	var deviceList []interface{} = []interface{}{}
	var after *permissions.ListAfter

	for found == limit {
		limit = 5
		query := permissions.QueryMessage{
			Resource: "devices",
			Find: &permissions.QueryFind{
				QueryListCommons: permissions.QueryListCommons{
					Offset:   0,
					Limit:    limit,
					After:    after,
					SortBy:   "id",
					SortDesc: true,
				},
				Filter: &permissions.Selection{
					Condition: permissions.ConditionConfig{
						Feature:   "features.owner_id",
						Value:     userId,
						Operation: permissions.QueryEqualOperation,
					},
				},
			}}
		res, code, err := c.permClient.Query(token, query)
		if err != nil {
			return result, err
		}
		if code != http.StatusOK {
			return result, errors.New("unexpected upstream status code")
		}
		if res == nil {
			return result, err
		}
		ok := false
		deviceList, ok = res.([]interface{})
		if !ok {
			return result, errUnexpectedReponseFormat
		}
		found = len(deviceList)

		deviceIds := []string{}
		deviceId := ""
		for _, device := range deviceList {
			deviceMap, ok := device.(map[string]interface{})
			if !ok {
				return result, errUnexpectedReponseFormat
			}
			deviceId, ok = deviceMap["id"].(string)
			if !ok {
				return result, errUnexpectedReponseFormat
			}
			deviceIds = append(deviceIds, deviceId)
		}
		after = &permissions.ListAfter{
			Id: deviceId,
		}

		usages, _, err := c.tsClient.GetDeviceUsage(token, deviceIds)
		if err != nil {
			return result, err
		}

		for _, usage := range usages {
			child := model.CostWithChildren{
				CostWithEstimation: model.CostWithEstimation{
					Month: model.CostEntry{
						Storage: pricingModel.Storage * float64(usage.Bytes) * float64(hoursInMonthProgressed) / 1000000000,
					},
				},
			}
			child.CostWithEstimation.EstimationMonth.Storage = extrapolateStorageUsage(usage, &pricingModel, &hoursInMonthTotal, &timeInMonthRemaining)
			result.Children[usage.DeviceId] = child
			result.CostWithEstimation.Month.Storage += child.Month.Storage
			result.CostWithEstimation.EstimationMonth.Storage += child.EstimationMonth.Storage
		}
	}

	return
}

func prepExtrapolate() (float64, int, time.Duration) {
	now := time.Now()
	hoursInMonthProgressed := 0
	hoursInMonthProgressed += (now.Day() - 1) * 24
	hoursInMonthProgressed += now.Hour()

	startNextMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	timeInMonthRemaining := startNextMonth.Sub(now)
	startThisMonth := time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC)
	hoursInMonthTotal := startNextMonth.Sub(startThisMonth).Hours()

	return hoursInMonthTotal, hoursInMonthProgressed, timeInMonthRemaining
}

func extrapolateStorageUsage(usage timescale_wrapper.Usage, pricingModel *model.PricingModel, hoursInMonthTotal *float64, timeInMonthRemaining *time.Duration) float64 {
	estimatedBytes := (float64(usage.Bytes) + (usage.BytesPerDay * timeInMonthRemaining.Hours() / 24.0))
	return pricingModel.Storage * estimatedBytes * (*hoursInMonthTotal) / 1000000000
}
