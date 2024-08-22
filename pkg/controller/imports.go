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
	"fmt"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
)

func (c *Controller) GetImportsTree(userId string, skipEstimation bool, start *time.Time, end *time.Time) (tree model.CostWithChildren, err error) {
	timer := time.Now()
	if (start == nil && end != nil) || (start != nil && end == nil) || (start != nil && !skipEstimation) {
		return tree, fmt.Errorf("must not provide only one of start or end. must not provide start and stop without skipEstimation")
	}
	filter := &statsFilter{
		CPU:     true,
		RAM:     true,
		Storage: false,
		filter: filter{
			Namespace: &c.config.NamespaceImports,
			Labels: map[string][]string{
				"label_user": {userId},
			},
			Start: start,
			End:   end,
		},
	}
	if !skipEstimation {
		filter.PredictionBasedOn = &d24h
	}
	stats, err := c.getStats(filter)
	if err != nil {
		return
	}

	tree = buildTree(stats, "label_import_id")
	c.logDebug("ImportsTree " + time.Since(timer).String())
	return
}
