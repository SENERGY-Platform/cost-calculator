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
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheus_model "github.com/prometheus/common/model"
	"golang.org/x/exp/maps"
)

type podStatsFilter struct {
	podFilter
	CPU               bool
	RAM               bool
	Storage           bool
	PredictionBasedOn *time.Duration
}

type podFilter struct {
	Namespace *string
	Labels    map[string][]string
}

type podStat struct {
	Labels prometheus_model.Metric
	model.CostWithEstimation
}

type upsertFlags struct {
	cpu               bool
	ram               bool
	storage           bool
	cpuEstimation     bool
	ramEstimation     bool
	storageEstimation bool
}

func (c *Controller) getPodsMonth(filter *podStatsFilter) (result []podStat, err error) {
	resultMap := map[string]podStat{}
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	var superErr error
	if filter.CPU {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cpustats, err := c.getPodCPUMonth(&filter.podFilter, filter.PredictionBasedOn)
			if err != nil {
				superErr = err
			}
			mux.Lock()
			defer mux.Unlock()
			err = upsertPodStats(cpustats, resultMap, &upsertFlags{cpu: true, cpuEstimation: true})
			if err != nil {
				superErr = err
			}
		}()
	}

	if filter.RAM {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ramstats, err := c.getPodRAMMonth(&filter.podFilter, filter.PredictionBasedOn)
			if err != nil {
				superErr = err
			}
			mux.Lock()
			defer mux.Unlock()
			err = upsertPodStats(ramstats, resultMap, &upsertFlags{ram: true, ramEstimation: true})
			if err != nil {
				superErr = err
			}
		}()
	}

	if filter.Storage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			storageStats, err := c.getPodStorageMonth(&filter.podFilter, filter.PredictionBasedOn)
			if err != nil {
				superErr = err
			}
			mux.Lock()
			defer mux.Unlock()
			err = upsertPodStats(storageStats, resultMap, &upsertFlags{storage: true, storageEstimation: true})
			if err != nil {
				superErr = err
			}
		}()
	}
	wg.Wait()
	if superErr != nil {
		return nil, superErr
	}

	result = maps.Values(resultMap)
	return
}

func (c *Controller) getPodCPUMonth(filter *podFilter, estimationBasedOn *time.Duration) (result []podStat, err error) {
	hoursInMonthProgressed, timeInMonthRemaining, hoursInMonthProgressedStr, _, _ := getMonthTimeInfo()

	baseQuery0 := "avg_over_time(namespace_pod_container:container_cpu_usage_seconds_total:avg_rate_1h{"
	if filter.Namespace != nil {
		baseQuery0 += "namespace=\"" + *filter.Namespace + "\""
	}
	baseQuery0 += "}"

	baseQuery1 := ") * on (namespace, pod) group_left(" + c.config.CustomPrometheusLabels + ") kube_pod_labels{container=\"kube-state-metrics\""
	if filter.Namespace != nil {
		baseQuery1 += ", namespace=\"" + *filter.Namespace + "\""
	}
	baseQuery1 += getLabelFilterStr(filter.Labels) + "}"

	promQuery := baseQuery0 + "[" + hoursInMonthProgressedStr + "h:]" + baseQuery1
	var promQueryPred *string
	if estimationBasedOn != nil {
		s := baseQuery0 + "[" + estimationBasedOn.String() + ":]" + baseQuery1
		promQueryPred = &s
	}
	return c.queryCpuRam(timeInMonthRemaining, hoursInMonthProgressed, &promQuery, promQueryPred, true)
}

func (c *Controller) getPodRAMMonth(filter *podFilter, estimationBasedOn *time.Duration) (result []podStat, err error) {
	hoursInMonthProgressed, timeInMonthRemaining, hoursInMonthProgressedStr, _, _ := getMonthTimeInfo()

	baseQuery0 := "avg_over_time(namespace_pod_container:container_memory_working_set_bytes:avg_1h"
	if filter.Namespace != nil {
		baseQuery0 += "{namespace=\"" + *filter.Namespace + "\"}"
	}

	baseQuery1 := ") * on (namespace, pod) group_left(" + c.config.CustomPrometheusLabels + ") kube_pod_labels{container=\"kube-state-metrics\""
	if filter.Namespace != nil {
		baseQuery1 += ", namespace=\"" + *filter.Namespace + "\""
	}
	baseQuery1 += getLabelFilterStr(filter.Labels) + "}"

	promQuery := baseQuery0 + "[" + hoursInMonthProgressedStr + "h:]" + baseQuery1

	var promQueryPred *string
	if estimationBasedOn != nil {
		s := baseQuery0 + "[" + estimationBasedOn.String() + ":]" + baseQuery1
		promQueryPred = &s
	}
	return c.queryCpuRam(timeInMonthRemaining, hoursInMonthProgressed, &promQuery, promQueryPred, false)
}

func (c *Controller) queryCpuRam(timeInMonthRemaining time.Duration, hoursInMonthProgressed int, promQuery *string, promQueryPred *string, isCpu bool) (result []podStat, err error) {
	if promQuery == nil {
		return result, fmt.Errorf("promQuery may not be null")
	}

	promResp, w, err := c.prometheus.Query(context.Background(), *promQuery, time.Now())
	if err != nil {
		return nil, err
	}
	values, err := validateAndGetValuesPromResponse(promResp, w)
	if err != nil {
		return nil, err
	}

	var estimationValues prometheus_model.Vector
	if promQueryPred != nil {
		promResp, w, err := c.prometheus.Query(context.Background(), *promQueryPred, time.Now())
		if err != nil {
			return nil, err
		}
		estimationValues, err = validateAndGetValuesPromResponse(promResp, w)
		if err != nil {
			return nil, err
		}
		slices.SortFunc(estimationValues, func(a, b *prometheus_model.Sample) int {
			return int(a.Metric.FastFingerprint() - b.Metric.FastFingerprint())
		})
	}

	for _, element := range values {
		stat := podStat{
			Labels: element.Metric,
			CostWithEstimation: model.CostWithEstimation{
				Month: model.CostEntry{},
			},
		}
		if isCpu {
			stat.CostWithEstimation.Month.Cpu = c.pricingModel.CPU * float64(element.Value) * float64(hoursInMonthProgressed)
		} else {
			stat.CostWithEstimation.Month.Ram = c.pricingModel.RAM * float64(element.Value) * float64(hoursInMonthProgressed) / 1000000000 // cost * avg-usage * hours-progressed  / correction-bytes-in-gb
		}
		if estimationValues != nil {
			i, ok := slices.BinarySearchFunc(estimationValues, element, func(a, b *prometheus_model.Sample) int {
				return int(a.Metric.FastFingerprint() - b.Metric.FastFingerprint())
			})
			stat.CostWithEstimation.EstimationMonth = model.CostEntry{}
			if isCpu {
				if ok {
					stat.CostWithEstimation.EstimationMonth.Cpu = stat.CostWithEstimation.Month.Cpu + c.pricingModel.CPU*float64(estimationValues[i].Value)*timeInMonthRemaining.Hours()
				} else {
					stat.CostWithEstimation.EstimationMonth.Cpu = stat.CostWithEstimation.Month.Cpu
				}
			} else {
				if ok {
					stat.CostWithEstimation.EstimationMonth.Ram = stat.CostWithEstimation.Month.Ram + c.pricingModel.RAM*float64(estimationValues[i].Value)*timeInMonthRemaining.Hours()/1000000000
				} else {
					stat.CostWithEstimation.EstimationMonth.Ram = stat.CostWithEstimation.Month.Ram
				}
			}
		}
		result = append(result, stat)
	}
	return

}

func (c *Controller) getPodStorageMonth(filter *podFilter, estimationBasedOn *time.Duration) (result []podStat, err error) {
	hoursInMonthProgressed, timeInMonthRemaining, hoursInMonthProgressedStr, _, _ := getMonthTimeInfo()

	result = []podStat{}
	promQuery := "avg_over_time"
	baseQuery0 := "(namespace_persistentvolumeclaim:kube_persistentvolumeclaim_resource_requests_storage_bytes:avg_1h{"
	if filter.Namespace != nil {
		baseQuery0 += "namespace=\"" + *filter.Namespace + "\""
	}

	baseQuery0 += "}"
	promQuery += baseQuery0 + "[" + hoursInMonthProgressedStr + "h:]"

	baseQuery1 := ") * on (namespace, persistentvolumeclaim) group_right() kube_pod_spec_volumes_persistentvolumeclaims_info{container=\"kube-state-metrics\""
	if filter.Namespace != nil {
		baseQuery1 += ", namespace=\"" + *filter.Namespace + "\""
	}
	baseQuery1 += "} * on (namespace, pod) group_left(" + c.config.CustomPrometheusLabels + ") kube_pod_labels{container=\"kube-state-metrics\""
	if filter.Namespace != nil {
		baseQuery1 += ", namespace=\"" + *filter.Namespace + "\""
	}
	baseQuery1 += getLabelFilterStr(filter.Labels) + "}"
	promQuery += baseQuery1
	promResp, w, err := c.prometheus.Query(context.Background(), promQuery, time.Now())
	if err != nil {
		return nil, err
	}
	values, err := validateAndGetValuesPromResponse(promResp, w)
	if err != nil {
		return nil, err
	}

	for _, element := range values {
		delete(element.Metric, "container") // This is always "kube-state-metrics" and should not be considered for container costs
		stat := podStat{
			Labels: element.Metric,
			CostWithEstimation: model.CostWithEstimation{
				Month: model.CostEntry{
					Storage: c.pricingModel.Storage * float64(element.Value) * float64(hoursInMonthProgressed) / 1000000000, // cost * avg-size * hours-progressed / correction-bytes-in-gb
				},
			},
		}
		if estimationBasedOn != nil {
			// Since we are calculating cost based on the PVC size and changes aren't common, just assume no changes and calculate cost based on time remaining
			stat.CostWithEstimation.EstimationMonth = model.CostEntry{}
			stat.CostWithEstimation.EstimationMonth.Storage = stat.CostWithEstimation.Month.Storage + c.pricingModel.Storage*float64(element.Value)*timeInMonthRemaining.Hours()/1000000000 // cost * avg-size * hours-progressed / correction-bytes-in-gb
		}
		result = append(result, stat)
	}
	return
}

func upsertPodStats(stats []podStat, m map[string]podStat, flags *upsertFlags) error {
	for _, stat := range stats {
		ns, ok := stat.Labels["namespace"]
		if !ok {
			return fmt.Errorf("missing namspace in labels %#v", stat.Labels)
		}
		pod, ok := stat.Labels["pod"]
		if !ok {
			return fmt.Errorf("missing pod in labels %#v", stat.Labels)
		}
		container, ok := stat.Labels["container"]
		if !ok {
			container = ""
		}
		key := string(ns) + string(pod) + string(container)
		entry, ok := m[key]
		if ok {
			if flags.cpu {
				entry.Month.Cpu = stat.Month.Cpu
			}
			if flags.ram {
				entry.Month.Ram = stat.Month.Ram
			}
			if flags.storage {
				entry.Month.Storage = stat.Month.Storage
			}
			if flags.cpuEstimation {
				entry.EstimationMonth.Cpu = stat.EstimationMonth.Cpu
			}
			if flags.ramEstimation {
				entry.EstimationMonth.Ram = stat.EstimationMonth.Ram
			}
			if flags.storageEstimation {
				entry.EstimationMonth.Storage = stat.EstimationMonth.Storage
			}
			for k, v := range stat.Labels {
				entry.Labels[k] = v
			}
		} else {
			entry = stat
		}
		m[key] = entry
	}
	return nil
}

func validateAndGetValuesPromResponse(promResp prometheus_model.Value, w v1.Warnings) (values prometheus_model.Vector, err error) {
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}
	if promResp.Type() != prometheus_model.ValVector {
		return nil, fmt.Errorf("unexpected prometheus response %#v", promResp)
	}
	values, ok := promResp.(prometheus_model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected prometheus response %#v", promResp)
	}
	return
}

func getLabelFilterStr(labels map[string][]string) string {
	if labels == nil {
		return ""
	}
	res := ""
	for k, v := range labels {
		if len(v) == 0 {
			continue
		}
		res += ", " + k + "="
		if len(v) == 1 {
			res += "\"" + v[0] + "\""
		} else {
			res += "~\"" + strings.Join(v, "|") + "\""
		}
	}
	return res
}
