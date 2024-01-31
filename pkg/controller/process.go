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
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
	prometheus_model "github.com/prometheus/common/model"
	"log"
	"slices"
	"strings"
	"time"
)

func (c *Controller) GetProcessTree(userId string) (result model.CostTree, err error) {
	controllers, err := c.GetCostControllersWithFilter(func(key string, allo opencost.AllocationEntry) (use bool, newName string) {
		return slices.Contains(c.config.ProcessCostSources, key) || slices.Contains(c.config.MarshallingCostSources, key), key
	})

	processCost := model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{
			Month:           model.CostEntry{},
			EstimationMonth: model.CostEntry{},
		},
		Children: map[string]model.CostWithChildren{},
	}

	marshallerCostTotal := model.CostWithEstimation{}

	start, end := getMonthTimeRange()

	userProcessFactor, err := c.getUserProcessFactor(userId, start, end)
	if err != nil {
		return result, err
	}

	for key, value := range controllers {
		if slices.Contains(c.config.ProcessCostSources, key) {
			nameParts := strings.Split(key, ":")
			name := nameParts[len(nameParts)-1]

			child := model.CostWithChildren{
				CostWithEstimation: model.CostWithEstimation{
					Month: model.CostEntry{
						Cpu:     value.Month.Cpu * userProcessFactor,
						Ram:     value.Month.Ram * userProcessFactor,
						Storage: value.Month.Storage * userProcessFactor,
					},
					EstimationMonth: model.CostEntry{
						Cpu:     value.EstimationMonth.Cpu * userProcessFactor,
						Ram:     value.EstimationMonth.Ram * userProcessFactor,
						Storage: value.EstimationMonth.Storage * userProcessFactor,
					},
				},
				Children: map[string]model.CostWithChildren{},
			}

			processCost.Children[name] = child

			processCost.Month.Cpu = processCost.Month.Cpu + child.Month.Cpu
			processCost.Month.Ram = processCost.Month.Ram + child.Month.Ram
			processCost.Month.Storage = processCost.Month.Storage + child.Month.Storage

			processCost.EstimationMonth.Cpu = processCost.EstimationMonth.Cpu + child.EstimationMonth.Cpu
			processCost.EstimationMonth.Ram = processCost.EstimationMonth.Ram + child.EstimationMonth.Ram
			processCost.EstimationMonth.Storage = processCost.EstimationMonth.Storage + child.EstimationMonth.Storage

			processDefinitionFactors, err := c.getProcessDefinitionFactors(key, userId, start, end)
			if err != nil {
				return result, err
			}
			for processDefinition, factor := range processDefinitionFactors {
				grandchild := model.CostWithChildren{
					CostWithEstimation: model.CostWithEstimation{
						Month: model.CostEntry{
							Cpu:     child.Month.Cpu * factor,
							Ram:     child.Month.Ram * factor,
							Storage: child.Month.Storage * factor,
						},
						EstimationMonth: model.CostEntry{
							Cpu:     child.EstimationMonth.Cpu * factor,
							Ram:     child.EstimationMonth.Ram * factor,
							Storage: child.EstimationMonth.Storage * factor,
						},
					},
					Children: map[string]model.CostWithChildren{},
				}
				child.Children[processDefinition] = grandchild
			}
		}

		if slices.Contains(c.config.MarshallingCostSources, key) {
			marshallerCostTotal.Month.Cpu = marshallerCostTotal.Month.Cpu + value.Month.Cpu
			marshallerCostTotal.Month.Ram = marshallerCostTotal.Month.Ram + value.Month.Ram
			marshallerCostTotal.Month.Storage = marshallerCostTotal.Month.Storage + value.Month.Storage

			marshallerCostTotal.EstimationMonth.Cpu = marshallerCostTotal.EstimationMonth.Cpu + value.EstimationMonth.Cpu
			marshallerCostTotal.EstimationMonth.Ram = marshallerCostTotal.EstimationMonth.Ram + value.EstimationMonth.Ram
			marshallerCostTotal.EstimationMonth.Storage = marshallerCostTotal.EstimationMonth.Storage + value.EstimationMonth.Storage
		}
	}

	processMarshallerFactor, err := c.getProcessMarshallerFactor(start, end)
	if err != nil {
		return result, err
	}

	userMarshallerFactor, err := c.getUserMarshallerFactor(userId, start, end)
	if err != nil {
		return result, err
	}

	marshallerCostProcesses := model.CostWithEstimation{
		Month: model.CostEntry{
			Cpu:     marshallerCostTotal.Month.Cpu * processMarshallerFactor,
			Ram:     marshallerCostTotal.Month.Ram * processMarshallerFactor,
			Storage: marshallerCostTotal.Month.Storage * processMarshallerFactor,
		},
		EstimationMonth: model.CostEntry{
			Cpu:     marshallerCostTotal.EstimationMonth.Cpu * processMarshallerFactor,
			Ram:     marshallerCostTotal.EstimationMonth.Ram * processMarshallerFactor,
			Storage: marshallerCostTotal.EstimationMonth.Storage * processMarshallerFactor,
		},
	}

	marshallerCostUser := model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{
			Month: model.CostEntry{
				Cpu:     marshallerCostProcesses.Month.Cpu * userMarshallerFactor,
				Ram:     marshallerCostProcesses.Month.Ram * userMarshallerFactor,
				Storage: marshallerCostProcesses.Month.Storage * userMarshallerFactor,
			},
			EstimationMonth: model.CostEntry{
				Cpu:     marshallerCostProcesses.EstimationMonth.Cpu * userMarshallerFactor,
				Ram:     marshallerCostProcesses.EstimationMonth.Ram * userMarshallerFactor,
				Storage: marshallerCostProcesses.EstimationMonth.Storage * userMarshallerFactor,
			},
		},
		Children: map[string]model.CostWithChildren{},
	}

	processCost.Children["marshalling"] = marshallerCostUser

	result = map[string]model.CostWithChildren{"process": processCost}
	return result, nil
}

func getMonthTimeRange() (start time.Time, end time.Time) {
	end = time.Now()
	y, m, _ := end.Date()
	start = time.Date(y, m, 1, 0, 0, 0, 0, end.Location())
	return
}

func (c *Controller) getUserProcessFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserProcessCostFractionQuery, userId, start, end)
}

func (c *Controller) getProcessMarshallerFactor(start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.ProcessMarshallerCostFractionQuery, "", start, end)
}

func (c *Controller) getUserMarshallerFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserMarshallerCostFractionQuery, userId, start, end)
}

func (c *Controller) getProcessDefinitionFactors(processCostSource string, userId string, start time.Time, end time.Time) (map[string]float64, error) {
	result := map[string]float64{}

	instanceId, ok := c.config.ProcessCostSourceToInstanceIdPlaceholderForProcessDefCostFraction[processCostSource]
	if !ok {
		return result, nil
	}

	query := strings.ReplaceAll(c.config.UserProcessDefinitionCostFractionQuery, "$instance_id", instanceId)

	ingreases, err := c.getValueMapFromPrometheus(query, userId, start, end)
	if err != nil {
		return result, err
	}
	sum := 0.0
	for _, e := range ingreases {
		sum = sum + e
	}
	for k, e := range ingreases {
		result[k] = e / sum
	}
	return result, nil
}

func (c *Controller) getValueFromPrometheus(query string, userId string, start time.Time, end time.Time) (float64, error) {
	query = strings.ReplaceAll(query, "$user_id", userId)
	query = strings.ReplaceAll(query, "$__range", end.Sub(start).Round(time.Second).String())
	resp, w, err := c.prometheus.Query(context.Background(), query, end)
	if err != nil {
		return 1, err
	}
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}
	if resp.Type() != prometheus_model.ValScalar {
		return 1, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	value, ok := resp.(*prometheus_model.Scalar)
	if !ok {
		return 1, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	return float64(value.Value), nil
}

func (c *Controller) getValueMapFromPrometheus(query string, userId string, start time.Time, end time.Time) (map[string]float64, error) {
	result := map[string]float64{}
	query = strings.ReplaceAll(query, "$user_id", userId)
	query = strings.ReplaceAll(query, "$__range", end.Sub(start).Round(time.Second).String())
	resp, w, err := c.prometheus.Query(context.Background(), query, end)
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
		label := ""
		for _, metricLabel := range element.Metric {
			label = string(metricLabel)
		}
		result[label] = float64(element.Value)
	}
	return result, nil
}
