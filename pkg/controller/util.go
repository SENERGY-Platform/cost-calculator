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
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	prometheus_model "github.com/prometheus/common/model"
)

const minutesInDay = 24 * 60

func predict(base model.CostEntry, l24h model.CostEntry) model.CostEntry {
	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	remainingMinutes := time.Until(nextMonth).Minutes()
	return model.CostEntry{
		Cpu:     base.Cpu + (remainingMinutes * (l24h.Cpu / minutesInDay)),
		Ram:     base.Ram + (remainingMinutes * (l24h.Ram / minutesInDay)),
		Storage: base.Storage + (remainingMinutes * (l24h.Storage / minutesInDay)),
	}
}

func calcStats(data []float64) (min, max, mean, median float64) {
	if len(data) == 0 {
		return 0, 0, 0, 0
	}
	slices.Sort(data)
	min = data[0]
	max = data[len(data)-1]
	if len(data)%2 == 0 {
		median = (data[len(data)/2] + data[len(data)/2-1]) / 2
	} else {
		median = data[len(data)/2]
	}
	var s float64 = 0
	for _, f := range data {
		s += f
	}
	mean = s / float64(len(data))
	return
}

func (c *Controller) getUsername(userId string) (username string, err error) {
	if userId == "" {
		return "", errors.New("No userId provided")
	}
	resp, err := http.Get(c.config.UserManagementUrl + "/user/id/" + userId + "/name")
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		return "", errors.New("unexpected upstream status code")
	}
	err = json.NewDecoder(resp.Body).Decode(&username)
	if err != nil {
		return
	}
	return
}

type costWithChildrenAndStats struct {
	stats    []podStat
	children map[string]costWithChildrenAndStats
	model.CostWithEstimation
}

func (c *costWithChildrenAndStats) toModelCostWithChildrenAndStats() (m model.CostWithChildren) {
	m = model.CostWithChildren{
		CostWithEstimation: c.CostWithEstimation,
		Children:           map[string]model.CostWithChildren{},
	}
	for k, v := range c.children {
		m.Children[k] = v.toModelCostWithChildrenAndStats()
	}
	return m
}

func buildTree(stats []podStat, labels ...string) (tree model.CostWithChildren) {
	t := _buildTree(stats, labels...)
	return t.toModelCostWithChildrenAndStats()
}

func _buildTree(stats []podStat, labels ...string) (tree costWithChildrenAndStats) {
	// make tree
	// put all stats in tree based on first label
	// for all remaining labels
	//	make tree based on that label
	tree = costWithChildrenAndStats{
		stats:    []podStat{},
		children: map[string]costWithChildrenAndStats{},
		CostWithEstimation: model.CostWithEstimation{
			Month:           model.CostEntry{},
			EstimationMonth: model.CostEntry{},
		},
	}
	if len(labels) == 0 {
		for _, child := range stats {
			tree.CostWithEstimation.Add(child.CostWithEstimation)
		}
		return tree
	}
	for _, stat := range stats {
		labelValue, ok := stat.Labels[prometheus_model.LabelName(labels[0])]
		if !ok {
			// Element can't be grouped further, add costs on this level
			tree.CostWithEstimation.Add(stat.CostWithEstimation)
			continue
		}
		children, ok := tree.children[string(labelValue)]
		if !ok {
			children = costWithChildrenAndStats{
				stats:    []podStat{},
				children: map[string]costWithChildrenAndStats{},
			}
		}
		children.stats = append(children.stats, stat)
		tree.children[string(labelValue)] = children
		tree.CostWithEstimation.Add(stat.CostWithEstimation)
	}
	for k, v := range tree.children {
		tree.children[k] = _buildTree(v.stats, labels[1:]...)
	}

	return tree
}
