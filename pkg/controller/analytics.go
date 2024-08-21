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
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
)

var d24h = time.Hour * 24

func (c *Controller) GetAnalyticsTree(userId string) (tree model.CostWithChildren, err error) {

	stats, err := c.getPodsMonth(&podStatsFilter{
		CPU:     true,
		RAM:     true,
		Storage: true,
		podFilter: podFilter{
			Namespace: &c.config.NamespaceAnalytics,
			Labels: map[string][]string{
				"label_user": {userId},
			},
		},
		PredictionBasedOn: &d24h,
	})
	if err != nil {
		return
	}

	tree = buildTree(stats, "label_pipeline_id", "pod", "container")

	return
}
