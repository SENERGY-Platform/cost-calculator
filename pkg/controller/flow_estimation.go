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

package controller

import (
	"fmt"
	"strings"
	"time"

	parsing_api "github.com/SENERGY-Platform/analytics-flow-engine/pkg/parsing-api"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

const cacheValid = 1 * time.Hour

func (c *Controller) GetFlowEstimations(authorization string, userid string, flowIds []string) (estimations []*model.Estimation, err error) {
	flows := []parsing_api.Pipeline{}
	allFlowsCached := true
	c.flowCacheMux.Lock()
	for _, flowId := range flowIds {
		// This is also an access check, don't cache across different users
		flow, err := c.parsingClient.GetPipeline(flowId, userid, authorization)
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
		if allFlowsCached {
			flowCached, ok := c.flowCache[flowId]
			if !ok || flowCached.enteredAt.Add(cacheValid).Before(time.Now()) {
				allFlowsCached = false
			}
		}
	}

	if allFlowsCached {
		for _, flowId := range flowIds {
			estimations = append(estimations, c.flowCache[flowId].estimation)
		}
		c.flowCacheMux.Unlock()
		return estimations, nil
	}
	c.flowCacheMux.Unlock()

	stats, err := c.getPodsMonth(&podStatsFilter{
		CPU:     true,
		RAM:     true,
		Storage: true,
		podFilter: podFilter{
			Namespace: &c.config.NamespaceAnalytics,
		},
		PredictionBasedOn: &d24h,
	})
	if err != nil {
		return nil, err
	}

	operatorStats := map[string][]float64{}
	flowStats := map[string][]float64{}
	for _, stat := range stats {
		containerName, ok := stat.Labels["container"]
		if !ok {
			flowId, ok := stat.Labels["label_flow_id"]
			if !ok {
				return nil, fmt.Errorf("stat is missing container and label_flow_id labels")
			}
			l, ok := operatorStats[string(flowId)]
			if !ok {
				l = []float64{}
			}
			l = append(l, stat.EstimationMonth.Cpu+stat.EstimationMonth.Ram+stat.EstimationMonth.Storage)
			flowStats[string(flowId)] = l
		} else {
			nameParts := strings.Split(string(containerName), "--")
			if len(nameParts) != 2 {
				return nil, fmt.Errorf("containerName is not formatted correctly %#v", containerName)
			}
			l, ok := operatorStats[nameParts[1]]
			if !ok {
				l = []float64{}
			}
			l = append(l, stat.EstimationMonth.Cpu+stat.EstimationMonth.Ram+stat.EstimationMonth.Storage)
			operatorStats[nameParts[1]] = l
		}
	}

	operatorEstimations := map[string]model.Estimation{}
	for id, stats := range operatorStats {
		min, max, mean, median := calcStats(stats)
		operatorEstimation := model.Estimation{Min: min, Max: max, Mean: mean, Median: median}
		operatorEstimations[id] = operatorEstimation
	}

	flowEstimations := map[string]model.Estimation{}
	for id, stats := range flowStats {
		min, max, mean, median := calcStats(stats)
		flowEstimation := model.Estimation{Min: min, Max: max, Mean: mean, Median: median}
		flowEstimations[id] = flowEstimation
	}

	for _, flow := range flows {
		estimation := &model.Estimation{}
		for _, operator := range flow.Operators {
			operatorEstimation, ok := operatorEstimations[operator.Id]
			if !ok {
				continue
			}

			estimation.Min += operatorEstimation.Min
			estimation.Max += operatorEstimation.Max
			estimation.Mean += operatorEstimation.Mean
			estimation.Median += operatorEstimation.Median
		}
		flowEstimation, ok := flowEstimations[flow.FlowId]
		if ok {
			estimation.Min += flowEstimation.Min
			estimation.Max += flowEstimation.Max
			estimation.Mean += flowEstimation.Mean
			estimation.Median += flowEstimation.Median
		}
		c.flowCacheMux.Lock()
		c.flowCache[flow.FlowId] = flowCacheEntry{
			estimation: estimation,
			enteredAt:  time.Now(),
		}
		c.flowCacheMux.Unlock()
		estimations = append(estimations, estimation)
	}
	return estimations, nil
}
