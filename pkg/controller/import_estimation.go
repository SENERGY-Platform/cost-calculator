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
	"strings"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

func (c *Controller) GetImportEstimation(authorization string, userid string, importTypeId string) (estimation *model.Estimation, err error) {
	stats, err := c.getStats(&statsFilter{
		CPU:     true,
		RAM:     true,
		Storage: false,
		filter: filter{
			Namespace: &c.config.NamespaceImports,
			Labels: map[string][]string{
				"label_import_type_id": {strings.ReplaceAll(importTypeId, ":", "_")},
			},
		},
		PredictionBasedOn: &d24h,
	})
	if err != nil {
		return nil, err
	}

	l := []float64{}
	for _, stat := range stats {
		l = append(l, stat.EstimationMonth.Cpu+stat.EstimationMonth.Ram+stat.EstimationMonth.Storage)
	}
	min, max, mean, median := calcMinMaxMeanMedian(l)
	return &model.Estimation{Min: min, Max: max, Mean: mean, Median: median}, nil
}
