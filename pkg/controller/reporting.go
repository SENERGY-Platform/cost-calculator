/*
 *    Copyright 2025 InfAI (CC SES)
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
	"math"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	prometheus_model "github.com/prometheus/common/model"
)

func (c *Controller) GetReportingTree(userId string, skipEstimation bool, start *time.Time, end *time.Time) (result model.CostWithChildren, err error) {
	timer := time.Now()

	if (start == nil && end != nil) || (start != nil && end == nil) || (start != nil && !skipEstimation) {
		return result, fmt.Errorf("must not provide only one of start or end. must not provide start and stop without skipEstimation")
	}
	if start == nil {
		start, end = defaultStartEnd()
	}
	result = model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{
			EstimationMonth: model.CostEntry{},
			Month:           model.CostEntry{},
		},
		Children: map[string]model.CostWithChildren{},
	}
	nextMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 0, 0, 0, 0, 0, time.UTC) // this is okay, because multiplier is only used in estimations, and estimations with start and stop set are not allowed
	multiplier := 1 / (float64(end.Sub(*start)) / float64(nextMonth.Sub(*start)))

	query := "round(sum by (report_id) (increase(reporting_queried_datapoints_tsdb_total{user_id=\"" + userId + "\"}[" + end.Sub(*start).Round(time.Second).String() + "]))) != 0"

	resp, w, err := c.prometheus.Query(context.Background(), query, *end)
	if err != nil {
		return result, err
	}
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}
	if resp.Type() != prometheus_model.ValVector {
		return result, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	values, ok := resp.(prometheus_model.Vector)
	if !ok {
		return result, fmt.Errorf("unexpected prometheus response %#v", resp)
	}

	for _, element := range values {
		reportId := ""
		if len(element.Metric) > 1 {
			return result, fmt.Errorf("unexpected prometheus response, metric should have length 1 %#v", resp)
		}
		for _, metricLabel := range element.Metric {
			reportId = string(metricLabel)
		}

		reportEntry := model.CostWithChildren{
			CostWithEstimation: model.CostWithEstimation{
				Month: model.CostEntry{},
			},
			Children: map[string]model.CostWithChildren{},
		}

		value := sampleToFloat(element.Value)

		reportEntry.CostWithEstimation.Month.Storage += value * c.pricingModel.ReportingDataPoint
		reportEntry.CostWithEstimation.Month.Requests += value
		result.Month.Storage += value * c.pricingModel.ReportingDataPoint
		result.Month.Requests += value

		if !skipEstimation {
			estimate := math.Round(value * multiplier)
			reportEntry.CostWithEstimation.EstimationMonth = model.CostEntry{}
			reportEntry.CostWithEstimation.EstimationMonth.Storage += estimate * c.pricingModel.ReportingDataPoint
			reportEntry.CostWithEstimation.EstimationMonth.Requests += estimate
			result.EstimationMonth.Storage += estimate * c.pricingModel.ReportingDataPoint
			result.EstimationMonth.Requests += estimate
		}

		result.Children[reportId] = reportEntry
	}
	c.logDebug("ReportingTree " + time.Since(timer).String())

	return
}
