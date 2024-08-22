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
	"regexp"
	"strconv"
	"strings"
	"time"

	serving "github.com/SENERGY-Platform/analytics-serving/client"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	"github.com/SENERGY-Platform/models/go/models"
	prometheus_model "github.com/prometheus/common/model"
)

/*	Limitations:
	- Export cost only considers storage cost.
*/

var exportTableMatch = regexp.MustCompile("userid:(.{22})_export:(.{22}).*")

func (c *Controller) GetExportsTree(userId string, token string, admin bool, skipEstimation bool, start *time.Time, end *time.Time) (result model.CostWithChildren, err error) {
	timer := time.Now()

	if (start == nil && end != nil) || (start != nil && end == nil) || (start != nil && !skipEstimation) {
		return result, fmt.Errorf("must not provide only one of start or end. must not provide start and stop without skipEstimation")
	}
	if start == nil {
		start, end = defaultStartEnd()
	}
	result = model.CostWithChildren{
		CostWithEstimation: model.CostWithEstimation{},
		Children:           map[string]model.CostWithChildren{},
	}

	var instances serving.Instances

	t := true
	options := serving.ListOptions{
		InternalOnly: &t,
	}
	if !admin {
		resp, err := c.servingClient.ListInstances(token, &options)
		if err != nil {
			return result, err
		}
		instances = resp.Instances
	} else {
		instances, err = c.servingClient.ListInstancesAsAdmin(token, &options)
		if err != nil {
			return
		}
	}

	shortUserId, err := models.ShortenId(userId)
	if err != nil {
		return result, err
	}

	tables := []string{}

	for _, instance := range instances {
		if instance.UserId != userId || instance.ExportDatabase.Url != c.config.ServingTimescaleConfiguredUrl {
			continue
		}
		id := instance.ID.String()

		shortId, err := models.ShortenId(id)
		if err != nil {
			return result, err
		}

		tables = append(tables, "userid:"+shortUserId+"_export:"+shortId)
	}

	durationPassed := end.Sub(*start).Round(time.Second)
	now := time.Now() // This is fine as getting a prediction and providing start and end times is not allowed
	endOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	durationRemaining := endOfMonth.Sub(now)

	tableSizeByteMap := map[string]float64{}

	insertWithQuery := func(promQuery string, estimation bool, ts time.Time) error {
		resp, w, err := c.prometheus.Query(context.Background(), promQuery, ts)
		if err != nil {
			return err
		}
		if len(w) > 0 {
			log.Printf("WARNING: prometheus warnings = %#v\n", w)
		}
		if resp.Type() != prometheus_model.ValVector {
			return fmt.Errorf("unexpected prometheus response %#v", resp)
		}
		values, ok := resp.(prometheus_model.Vector)
		if !ok {
			return fmt.Errorf("unexpected prometheus response %#v", resp)
		}

		for _, element := range values {
			table, ok := element.Metric["table"]
			if !ok {
				return fmt.Errorf("unexpected prometheus response element %#v", element)
			}

			matches := exportTableMatch.FindAllStringSubmatch(string(table), -1)
			if matches == nil || len(matches[0]) != 3 {
				return fmt.Errorf("received metric for unexpected table name %#v", table)
			}

			exportId, err := models.LongId(matches[0][2])
			if err != nil {
				return err
			}

			tableSizeBytes := sampleToFloat(element.Value)
			child, ok := result.Children[string(exportId)]
			if !ok {
				child = model.CostWithChildren{
					CostWithEstimation: model.CostWithEstimation{
						Month:           model.CostEntry{},
						EstimationMonth: model.CostEntry{},
					},
				}
			}
			if estimation {
				tableSizeBytesEstimation := tableSizeBytes
				tableSizeBytes, ok := tableSizeByteMap[exportId]
				if !ok {
					tableSizeBytes = 0
				}

				avgFutureTableSize := (tableSizeBytesEstimation + tableSizeBytes) / 2
				futureCost := c.pricingModel.Storage * avgFutureTableSize * durationRemaining.Hours() / 1000000000 // cost * avg-size * hours-progressed / correction-bytes-in-gb
				child.CostWithEstimation.EstimationMonth.Storage = child.CostWithEstimation.Month.Storage + futureCost
				result.CostWithEstimation.EstimationMonth.Storage += child.EstimationMonth.Storage
			} else {
				tableSizeByteMap[exportId] = tableSizeBytes
				child.CostWithEstimation.Month.Storage = c.pricingModel.Storage * tableSizeBytes * durationPassed.Hours() / 1000000000 // cost * avg-size * hours-progressed / correction-bytes-in-gb
				result.CostWithEstimation.Month.Storage += child.Month.Storage
			}
			result.Children[exportId] = child
		}
		return nil
	}
	// Costs in current month
	promQuery := "avg_over_time(avg by (table) (timescale_table_size_bytes{table=~\"" + strings.Join(tables, "|") + "\"})[" + durationPassed.String() + ":])"
	err = insertWithQuery(promQuery, false, *end)
	if err != nil {
		return result, err
	}

	// Estimations
	if !skipEstimation {
		promQuery = "predict_linear(avg by (table) (timescale_table_size_bytes{table=~\"" + strings.Join(tables, "|") + "\"})[24h:], " + strconv.FormatFloat(durationRemaining.Seconds(), 'f', 0, 64) + ")"
		err = insertWithQuery(promQuery, true, time.Now())
		if err != nil {
			return result, err
		}
	}
	c.logDebug("ExportsTree " + time.Since(timer).String())
	return
}
