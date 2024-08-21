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
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	prometheus_model "github.com/prometheus/common/model"
)

func (c *Controller) GetProcessTree(userId string) (processCost model.CostWithChildren, err error) {
	timer := time.Now()
	processCost = model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{
			Month:           model.CostEntry{},
			EstimationMonth: model.CostEntry{},
		},
		Children: map[string]model.CostWithChildren{},
	}

	start, end := getMonthTimeRange()
	timer2 := time.Now()
	userProcessFactor, err := c.getUserProcessFactor(userId, start, end)
	if err != nil {
		return processCost, err
	}
	c.logDebug("ProcessTree: getUserProcessFactor " + time.Since(timer2).String())

	if userProcessFactor > 0 {
		for k, v := range c.config.ProcessCostSources {
			stats, err := c.getPodsMonth(&podStatsFilter{
				CPU:     true,
				RAM:     true,
				Storage: true,
				podFilter: podFilter{
					Namespace: &k,
					Labels: map[string][]string{
						"pod": v,
					},
				},
				PredictionBasedOn: &d24h,
			})
			if err != nil {
				return processCost, err
			}
			for _, stat := range stats {
				nameLabel, ok := stat.Labels["pod"]
				if !ok {
					return processCost, errors.New("missing label pod")
				}
				name := string(nameLabel)
				nameParts := strings.Split(string(nameLabel), "-")
				i := len(nameParts)
				if regexp.MustCompile(`.*-\d+$`).Match([]byte(name)) {
					// is stateful set pod, they always end in -\d
					i -= 1
				} else {
					// is something else, always ends in -xxxxxxxxx-xxxxx
					i -= 2
				}
				name = strings.Join(nameParts[:i], "-")

				child := model.CostWithChildren{
					CostWithEstimation: model.CostWithEstimation{
						Month: model.CostEntry{
							Cpu:     stat.Month.Cpu * userProcessFactor,
							Ram:     stat.Month.Ram * userProcessFactor,
							Storage: stat.Month.Storage * userProcessFactor,
						},
						EstimationMonth: model.CostEntry{
							Cpu:     stat.EstimationMonth.Cpu * userProcessFactor,
							Ram:     stat.EstimationMonth.Ram * userProcessFactor,
							Storage: stat.EstimationMonth.Storage * userProcessFactor,
						},
					},
					Children: map[string]model.CostWithChildren{},
				}
				existingChild, ok := processCost.Children[name]
				if ok {
					existingChild.Add(child.CostWithEstimation)
					processCost.Children[name] = existingChild
				} else {
					processCost.Children[name] = child
				}

				processCost.Month.Cpu = processCost.Month.Cpu + child.Month.Cpu
				processCost.Month.Ram = processCost.Month.Ram + child.Month.Ram
				processCost.Month.Storage = processCost.Month.Storage + child.Month.Storage

				processCost.EstimationMonth.Cpu = processCost.EstimationMonth.Cpu + child.EstimationMonth.Cpu
				processCost.EstimationMonth.Ram = processCost.EstimationMonth.Ram + child.EstimationMonth.Ram
				processCost.EstimationMonth.Storage = processCost.EstimationMonth.Storage + child.EstimationMonth.Storage

				processDefinitionFactors, err := c.getProcessDefinitionFactors(name, userId, start, end)
				if err != nil {
					return processCost, err
				}
				for processDefinition, factor := range processDefinitionFactors {
					if factor == 0 {
						continue
					}
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
		}
	}

	marshallerCostTotal := model.CostWithEstimation{}
	for k, v := range c.config.MarshallingCostSources {
		stats, err := c.getPodsMonth(&podStatsFilter{
			CPU:     true,
			RAM:     true,
			Storage: true,
			podFilter: podFilter{
				Namespace: &k,
				Labels: map[string][]string{
					"pod": v,
				},
			},
			PredictionBasedOn: &d24h,
		})
		if err != nil {
			return processCost, err
		}
		tree := buildTree(stats, "namespace")
		marshallerCostTotal.Add(tree.CostWithEstimation)
	}

	timer2 = time.Now()
	userMarshallerFactor, err := c.getUserMarshallerFactor(userId, start, end)
	if err != nil {
		return processCost, err
	}
	c.logDebug("ProcessTree: getUserMarshallerFactor " + time.Since(timer2).String())

	if userMarshallerFactor > 0 {
		timer2 = time.Now()
		processMarshallerFactor, err := c.getProcessMarshallerFactor(start, end)
		if err != nil {
			return processCost, err
		}
		c.logDebug("ProcessTree: getProcessMarshallerFactor " + time.Since(timer2).String())

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
	}

	timer2 = time.Now()
	userProcessIoFactor, err := c.getUserProcessIoFactor(userId, start, end)
	if err != nil {
		return processCost, err
	}
	c.logDebug("ProcessTree: getUserProcessIoFactor " + time.Since(timer2).String())
	if userProcessIoFactor != 0 {
		processIoCostTotal := model.CostWithEstimation{}
		for k, v := range c.config.ProcessIoCostSources {
			stats, err := c.getPodsMonth(&podStatsFilter{
				CPU:     true,
				RAM:     true,
				Storage: true,
				podFilter: podFilter{
					Namespace: &k,
					Labels: map[string][]string{
						"pod": v,
					},
				},
				PredictionBasedOn: &d24h,
			})
			if err != nil {
				return processCost, err
			}
			tree := buildTree(stats, "namespace")
			processIoCostTotal.Add(tree.CostWithEstimation)
		}

		processIoCostUser := model.CostWithChildren{
			CostWithEstimation: model.CostWithEstimation{
				Month: model.CostEntry{
					Cpu:     processIoCostTotal.Month.Cpu * userProcessIoFactor,
					Ram:     processIoCostTotal.Month.Ram * userProcessIoFactor,
					Storage: processIoCostTotal.Month.Storage * userProcessIoFactor,
				},
				EstimationMonth: model.CostEntry{
					Cpu:     processIoCostTotal.EstimationMonth.Cpu * userProcessIoFactor,
					Ram:     processIoCostTotal.EstimationMonth.Ram * userProcessIoFactor,
					Storage: processIoCostTotal.EstimationMonth.Storage * userProcessIoFactor,
				},
			},
			Children: map[string]model.CostWithChildren{},
		}
		processCost.Children["process-io"] = processIoCostUser
	}
	c.logDebug("ProcessTree " + time.Since(timer).String())

	return processCost, nil
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

func (c *Controller) getUserProcessIoFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserProcessIoCostFractionQuery, userId, start, end)
}

func (c *Controller) getProcessDefinitionFactors(processCostSource string, userId string, start time.Time, end time.Time) (map[string]float64, error) {
	result := map[string]float64{}

	instanceId, ok := c.config.ProcessCostSourceToInstanceIdPlaceholderForProcessDefCostFraction[processCostSource]
	if !ok {
		return result, nil
	}

	query := strings.ReplaceAll(c.config.UserProcessDefinitionCostFractionQuery, "$instance_id", instanceId)

	increases, err := c.getValueMapFromPrometheus(query, userId, start, end)
	if err != nil {
		return result, err
	}
	sum := 0.0
	for _, e := range increases {
		sum = sum + e
	}
	for k, e := range increases {
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
	return sampleToFloat(value.Value), nil
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
		result[label] = sampleToFloat(element.Value)
	}
	return result, nil
}

func sampleToFloat(value prometheus_model.SampleValue) float64 {
	temp := float64(value)
	if math.IsNaN(temp) {
		return 0
	}
	return temp
}
