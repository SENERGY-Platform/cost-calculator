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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SENERGY-Platform/device-repository/lib/client"

	"github.com/SENERGY-Platform/cost-calculator/pkg/log"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	"github.com/SENERGY-Platform/models/go/models"
	prometheus_model "github.com/prometheus/common/model"
)

var errUnexpectedReponseFormat = errors.New("unexpected response format")
var deviceTableMatch = regexp.MustCompile("device:(.{22})_service:(.{22}).*")

const deviceIdPrefix = "urn:infai:ses:device:"

/*	Limitations:
	- Device cost only considers storage cost.
*/

func (c *Controller) GetDevicesTree(ctx context.Context, userId string, token string, skipEstimation bool, start *time.Time, end *time.Time) (result model.CostWithChildren, err error) {
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

	var limit int64 = 0
	var offset int64 = 0
	var found int64 = 0

	deviceList := []models.Device{}
	for found == limit {
		limit = 5000
		deviceList, err, _ = c.deviceRepo.ListDevices(token, client.DeviceListOptions{
			Owner:  userId,
			Limit:  limit,
			Offset: offset,
			SortBy: "name.asc",
		})
		if err != nil {
			return result, err
		}
		found = int64(len(deviceList))
		offset = offset + limit
		tables := []string{}
		deviceIds := []string{}
		for _, device := range deviceList {
			deviceIds = append(deviceIds, device.Id)
			shortDeviceId, err := models.ShortenId(device.Id)
			if err != nil {
				return result, err
			}
			tables = append(tables, "device:"+shortDeviceId+".*")
		}

		tableSizeByteMap := map[string]float64{}

		insertWithQuery := func(promQuery string, metricName prometheus_model.LabelName, ts time.Time, callback func(metricValue string, value float64, child *model.CostWithChildren)) error {
			resp, w, err := c.prometheus.Query(ctx, promQuery, ts)
			if err != nil {
				return err
			}
			if len(w) > 0 {
				log.Logger.Warn("prometheus warnings", "warnings", w)
			}
			if resp.Type() != prometheus_model.ValVector {
				return fmt.Errorf("unexpected prometheus response %#v", resp)
			}
			values, ok := resp.(prometheus_model.Vector)
			if !ok {
				return fmt.Errorf("unexpected prometheus response %#v", resp)
			}

			for _, element := range values {
				metric, ok := element.Metric[metricName]
				if !ok {
					return fmt.Errorf("unexpected prometheus response element %#v", element)
				}
				metricStr := string(metric)

				id := metricStr
				if !strings.HasPrefix(metricStr, deviceIdPrefix) {
					matches := deviceTableMatch.FindAllStringSubmatch(metricStr, -1)
					if matches == nil || len(matches[0]) != 3 {
						return fmt.Errorf("received metric for unexpected table name %#v", metricStr)
					}

					id, err = models.LongId(matches[0][1])
					if err != nil {
						return err
					}

					id = deviceIdPrefix + id
				}

				val := sampleToFloat(element.Value)
				child, ok := result.Children[string(id)]
				if !ok {
					child = model.CostWithChildren{
						CostWithEstimation: model.CostWithEstimation{
							Month:           model.CostEntry{},
							EstimationMonth: model.CostEntry{},
						},
					}
				}
				callback(metricStr, val, &child)
				result.Children[id] = child
			}
			return nil
		}

		durationPassed := end.Sub(*start).Round(time.Second)
		now := time.Now() // This is fine as getting a prediction and providing start and end times is not allowed
		endOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		durationRemaining := endOfMonth.Sub(now)

		// Costs in current month
		timer2 := time.Now()
		promQuery := "avg_over_time(table:timescale_table_size_bytes:avg_1h{table=~\"" + strings.Join(tables, "|") + "\"}[" + durationPassed.String() + ":])"
		err = insertWithQuery(promQuery, "table", *end, func(table string, value float64, child *model.CostWithChildren) {
			tableSizeBytesEstimation := value
			tableSizeBytes, ok := tableSizeByteMap[table]
			if !ok {
				tableSizeBytes = 0
			}

			if !skipEstimation {
				avgFutureTableSize := (tableSizeBytesEstimation + tableSizeBytes) / 2
				futureCost := c.pricingModel.Storage * avgFutureTableSize * durationRemaining.Hours() / 1000000000 // cost * avg-size * hours-progressed / correction-bytes-in-gb
				child.CostWithEstimation.EstimationMonth.Storage += futureCost
				result.CostWithEstimation.EstimationMonth.Storage += futureCost
			}
		})
		if err != nil {
			return result, err
		}
		c.logDebug("DevicesTree: Current Month " + time.Since(timer2).String())

		// Estimations
		if !skipEstimation {
			timer2 = time.Now()
			promQuery = "predict_linear(table:timescale_table_size_bytes:avg_1h{table=~\"" + strings.Join(tables, "|") + "\"}[24h:], " + strconv.FormatFloat(durationRemaining.Seconds(), 'f', 0, 64) + ")"
			err = insertWithQuery(promQuery, "table", time.Now(), func(table string, value float64, child *model.CostWithChildren) {
				existingTableSizeBytes, ok := tableSizeByteMap[table]
				if !ok {
					existingTableSizeBytes = 0
				}
				tableSizeByteMap[table] = existingTableSizeBytes + value
				additionalCost := c.pricingModel.Storage * value * durationPassed.Hours() / 1000000000 // cost * avg-size * hours-remaining / correction-bytes-in-gb
				child.CostWithEstimation.Month.Storage += additionalCost
				result.CostWithEstimation.Month.Storage += additionalCost
				child.CostWithEstimation.EstimationMonth.Storage += additionalCost
				result.CostWithEstimation.EstimationMonth.Storage += additionalCost
			})
			if err != nil {
				return result, err
			}
			c.logDebug("DevicesTree: Estimations " + time.Since(timer2).String())
		}

		// Requests
		timer2 = time.Now()
		nextMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 0, 0, 0, 0, 0, time.UTC) // this is okay, because multiplier is only used in estimations, and estimations with start and stop set are not allowed
		multiplier := 1 / (float64(end.Sub(*start)) / float64(nextMonth.Sub(*start)))
		promQuery = "round(sum_over_time(device_id:connector_source_received_device_msg_size_count:sum_increase_1h{device_id=~\"" + strings.Join(deviceIds, "|") + "\"}[" + durationPassed.String() + "])) != 0"
		err = insertWithQuery(promQuery, "device_id", *end, func(table string, value float64, child *model.CostWithChildren) {
			child.Month.Requests = value
			result.CostWithEstimation.Month.Requests += child.Month.Requests
			if !skipEstimation {
				child.EstimationMonth.Requests = child.Month.Requests * multiplier
				result.CostWithEstimation.EstimationMonth.Requests += child.EstimationMonth.Requests
			}
		})
		if err != nil {
			return result, err
		}
		c.logDebug("DevicesTree: Requests " + time.Since(timer2).String())
	}
	c.logDebug("DevicesTree " + time.Since(timer).String())
	return
}
