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
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	serving "github.com/SENERGY-Platform/analytics-serving/client"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	prometheus_model "github.com/prometheus/common/model"
)

/*	Limitations:
	- Export cost only considers storage cost.
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

	_, hoursInMonthProgressed, timeInMonthRemaining := prepExtrapolate()
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

	shortUserId, err := models.ShortenId(userId)
	if err != nil {
		return result, err
	}

	hoursInMonthProgressedStr := strconv.Itoa(hoursInMonthProgressed)
	secondsInMonthRemainingStr := strconv.Itoa(int(timeInMonthRemaining.Seconds()))

	for _, instance := range instances {
		if instance.UserId != userId || instance.ExportDatabase.Url != c.config.ServingTimescaleConfiguredUrl {
			continue
		}
		id := instance.ID.String()

		shortId, err := models.ShortenId(id)
		if err != nil {
			return result, err
		}
		query := "sum(avg_over_time(timescale_table_size_bytes{table=\"userid:" + shortUserId + "_export:" + shortId + "\"}[" + hoursInMonthProgressedStr + "h]))"
		tableSizeBytes, err := c.getSinglePrometheusValue(query)
		if err != nil {
			return result, err
		}

		child := model.CostWithChildren{
			CostWithEstimation: model.CostWithEstimation{
				Month: model.CostEntry{
					Storage: pricingModel.Storage * tableSizeBytes * float64(hoursInMonthProgressed) / 1000000000, // cost * avg-size * hours-progressed / correction-bytes-in-gb
				},
			},
		}

		query = "sum(predict_linear(timescale_table_size_bytes{table=\"userid:" + shortUserId + "_export:" + shortId + "\"}[24h], " + secondsInMonthRemainingStr + "))"
		tableSizeBytesEstimation, err := c.getSinglePrometheusValue(query)
		if err != nil {
			return result, err
		}

		avgFutureTableSize := (tableSizeBytesEstimation + tableSizeBytes) / 2
		futureCost := pricingModel.Storage * avgFutureTableSize * timeInMonthRemaining.Hours() / 1000000000 // cost * avg-size * hours-progressed / correction-bytes-in-gb

		child.CostWithEstimation.EstimationMonth.Storage = child.CostWithEstimation.Month.Storage + futureCost
		result.Children[id] = child
		result.CostWithEstimation.Month.Storage += child.Month.Storage
		result.CostWithEstimation.EstimationMonth.Storage += child.EstimationMonth.Storage

	}

	return
}

func (c *Controller) getSinglePrometheusValue(query string) (result float64, err error) {
	promRes, w, err := c.prometheus.Query(context.Background(), query, time.Now())
	if err != nil {
		return result, err
	}
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}

	if promRes.Type() != prometheus_model.ValVector {
		return result, fmt.Errorf("unexpected prometheus response %#v", promRes)
	}

	values, ok := promRes.(prometheus_model.Vector)
	if !ok {
		return result, fmt.Errorf("unexpected prometheus response %#v", promRes)
	}

	if len(values) != 1 {
		log.Printf("WARNING: empty result for prometheus query, returning 0. Query: %#v", query)
		return 0, nil
	}

	result = sampleToFloat(values[0].Value)
	return
}
