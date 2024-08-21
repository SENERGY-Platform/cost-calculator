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
	"encoding/json"
	"testing"
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/configuration"
)

func TestGetCostTree(t *testing.T) {
	t.Skip("experiment")
	t.Log("must be manually evaluated")
	t.Log("expects port forwarding to prometheus: kubectl port-forward -n cattle-monitoring-system service/prometheus-operated 9090:9090")

	userId := "dd69ea0d-f553-4336-80f3-7f4567f85c7b" //replace with other examples

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.PrometheusUrl = "http://localhost:9090"

	ctrl, err := NewController(context.Background(), config, func(err error) {
		t.Fatal(err)
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	result, err := ctrl.GetProcessTree(userId)
	if err != nil {
		t.Error(err)
		return
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%#v\n%v\n", result, string(b))

}

func TestGetUserProcessFactor(t *testing.T) {
	t.Skip("experiment")
	t.Log("must be manually evaluated")
	t.Log("expects port forwarding: kubectl port-forward -n cattle-monitoring-system service/prometheus-operated 9090:9090")
	userId := "dd69ea0d-f553-4336-80f3-7f4567f85c7b" //replace with other examples

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.PrometheusUrl = "http://localhost:9090"

	ctrl, err := NewController(context.Background(), config, func(err error) {
		t.Fatal(err)
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	start, end := getMonthTimeRange()

	t.Log(ctrl.getUserProcessFactor(userId, start, end))
}

func TestGetProcessDefinitionFactor(t *testing.T) {
	t.Skip("experiment")
	t.Log("must be manually evaluated")
	t.Log("expects port forwarding: kubectl port-forward -n cattle-monitoring-system service/prometheus-operated 9090:9090")
	userId := "dd69ea0d-f553-4336-80f3-7f4567f85c7b" //replace with other examples

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.PrometheusUrl = "http://localhost:9090"

	ctrl, err := NewController(context.Background(), config, func(err error) {
		t.Fatal(err)
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	start, end := getMonthTimeRange()

	t.Log(ctrl.getProcessDefinitionFactors("__unallocated__/process-task-worker/deployment:pessimistic-worker", userId, start, end))
}

func TestGetProcessDefinitionFactorFactor(t *testing.T) {
	t.Skip("experiment")
	t.Log("must be manually evaluated")
	t.Log("expects port forwarding: kubectl port-forward -n cattle-monitoring-system service/prometheus-operated 9090:9090")
	userId := "dd69ea0d-f553-4336-80f3-7f4567f85c7b" //replace with other examples

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.PrometheusUrl = "http://localhost:9090"

	ctrl, err := NewController(context.Background(), config, func(err error) {
		t.Fatal(err)
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	//start, end := getMonthTimeRange()
	end := time.Now()
	start := end.Add(-24 * time.Hour)

	t.Log(ctrl.getValueMapFromPrometheus("sum( increase(external_task_worker_task_command_send_count_vec[$__range]) ) by (process_definition_id)", userId, start, end))
}
