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
	serving "github.com/SENERGY-Platform/analytics-serving/client"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
)

/*	Limitations:
	- Export cost only considers storage cost.
	- Storage cost is calculated as the storage used at the end of the
	  month as if this was the storage used during the whole month.
	  Any changes in usage over the month will be disregarded.
*/

const exportInstancePermissionsTopic = "export-instances"

func (c *Controller) GetExportsTree(userId string, token string, admin bool) (result model.CostWithChildren, err error) {
	result = model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{},
		Children:           map[string]model.CostWithChildren{},
	}

	pricingModel, err := c.opencost.GetPricingModel()
	if err != nil {
		return result, err
	}

	hoursInMonthTotal, hoursInMonthProgressed, timeInMonthRemaining := prepExtrapolate()
	var instances serving.Instances

	t := true
	options := serving.ListOptions{
		InternalOnly: &t,
	}
	if !admin {
		resp, err := c.servingClient.ListInstances(token, &options)
		if err != nil {
			return result, err
		}
		instances = resp.Instances
	} else {
		instances, err = c.servingClient.ListInstancesAsAdmin(token, &options)
		if err != nil {
			return
		}
	}

	exportIds := []string{}

	for _, instance := range instances {
		if instance.UserId == userId {
			exportIds = append(exportIds, instance.ID.String())
		}
	}

	usages, _, err := c.tsClient.GetExportUsage(token, exportIds)
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
		result.Children[usage.ExportId] = child
		result.CostWithEstimation.Month.Storage += child.Month.Storage
		result.CostWithEstimation.EstimationMonth.Storage += child.EstimationMonth.Storage
	}

	return
}
