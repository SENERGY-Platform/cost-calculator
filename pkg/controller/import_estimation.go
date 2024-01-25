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
	"regexp"
	"strings"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
)

func (c *Controller) GetImportEstimation(authorization string, userid string, importTypeId string) (estimation *model.Estimation, err error) {
	containerEntries, err := c.getCostContainers24hWithAggregate("", model.CostTypeImports, "", "label:user,namespace,label:importTypeId,container")
	if err != nil {
		return nil, err
	}
	rgx, err := regexp.Compile(strings.ReplaceAll(importTypeId, ":", "_") + "/import-.*")
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
	estimation = &model.Estimation{Min: min * daysInMonth, Max: max * daysInMonth, Mean: mean * daysInMonth, Median: median * daysInMonth}

	return estimation, nil
}
