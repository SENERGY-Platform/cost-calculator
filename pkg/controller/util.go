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
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
)

const cacheValid = time.Duration(15 * time.Minute)
const minutesInDay = 24 * 60

func validateAllocation(allocation *opencost.AllocationResponse) error {
	if allocation.Code != http.StatusOK {
		return errors.New("unexpected upstream statuscode")
	}
	if len(allocation.Data) != 1 {
		return errors.New("unexpected upstream response")
	}
	return nil
}

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

func getPrefetchFunction(window, aggregate, key string) func(c *Controller) error {
	return func(c *Controller) error {
		allocation, err := c.opencost.Allocation(&opencost.AllocationOptions{
			Window:    window,
			Aggregate: aggregate,
		})
		if err != nil {
			log.Println("WARNING: Could not prefetch: " + err.Error())
			return err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			log.Println("WARNING: Could not prefetch, invalid allocation response: " + err.Error())
			return err
		}
		c.cacheMux.Lock()
		defer c.cacheMux.Unlock()
		c.cache[key] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	}
}
