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
	"regexp"
	"time"

	parsing_api "github.com/SENERGY-Platform/analytics-flow-engine/pkg/parsing-api"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
)

const daysInMonth = 30

func (c *Controller) GetFlowEstimation(authorization string, userid string, flowId string) (estimation *model.Estimation, err error) {
	var flow parsing_api.Pipeline
	c.flowCacheMux.Lock()
	pipelineCached, ok := c.flowCache[flowId]
	c.flowCacheMux.Unlock()
	if ok && pipelineCached.enteredAt.Add(cacheValid).After(time.Now()) {
		flow = pipelineCached.flow
	} else {
		flow, err = c.parsingClient.GetPipeline(flowId, userid, authorization)
		if err != nil {
			return nil, err
		}
		c.flowCacheMux.Lock()
		c.flowCache[flowId] = flowCacheEntry{flow: flow, enteredAt: time.Now()}
		c.flowCacheMux.Unlock()
	}
	containerEntries, err := c.getCostContainers24h("", model.CostTypeAnalytics, "")
	if err != nil {
		return nil, err
	}
	estimation = &model.Estimation{}
	for _, operator := range flow.Operators {
		c.operatorCacheMux.Lock()
		cached, ok := c.operatorCache[operator.OperatorId]
		c.operatorCacheMux.Unlock()
		var operatorEstimation model.Estimation
		if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
			operatorEstimation = cached.estimation
		} else {
			rgx, err := regexp.Compile("deployment:pipeline-.{37}" + operator.OperatorId + "--.*")
			if err != nil {
				return nil, err
			}
			operatorEntries := []float64{}
			for key, containerEntry := range containerEntries {
				if rgx.MatchString(key) {
					operatorEntries = append(operatorEntries, containerEntry.Cpu+containerEntry.Ram+containerEntry.Storage)
				}
			}
			min, max, mean, median := calcStats(operatorEntries)
			operatorEstimation = model.Estimation{Min: min * daysInMonth, Max: max * daysInMonth, Mean: mean * daysInMonth, Median: median * daysInMonth}
			c.operatorCacheMux.Lock()
			c.operatorCache[operator.OperatorId] = operatorCacheEntry{estimation: *estimation, enteredAt: time.Now()}
			c.operatorCacheMux.Unlock()
		}

		estimation.Min += operatorEstimation.Min
		estimation.Max += operatorEstimation.Max
		estimation.Mean += operatorEstimation.Mean
		estimation.Median += operatorEstimation.Median
	}
	return estimation, nil
}
