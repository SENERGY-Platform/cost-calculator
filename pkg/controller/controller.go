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
	"context"
	"log"
	"sync"
	"time"

	parsing_api "github.com/SENERGY-Platform/analytics-flow-engine/pkg/parsing-api"
	serving "github.com/SENERGY-Platform/analytics-serving/client"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/configuration"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/model"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
	permissions "github.com/SENERGY-Platform/permission-search/lib/client"
	timescale_wrapper "github.com/SENERGY-Platform/timescale-wrapper/pkg/client"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var prefetchFn = []func(c *Controller) error{}

type cacheEntry struct {
	allocation opencost.AllocationResponse
	enteredAt  time.Time
}

type operatorCacheEntry struct {
	estimation model.Estimation
	enteredAt  time.Time
}

type flowCacheEntry struct {
	flow      parsing_api.Pipeline
	enteredAt time.Time
}

type Controller struct {
	opencost      *opencost.Client
	config        configuration.Config
	cache         map[string]cacheEntry
	cacheMux      sync.Mutex
	parsingClient *parsing_api.ParsingApi

	operatorCache    map[string]operatorCacheEntry
	operatorCacheMux sync.Mutex

	flowCache    map[string]flowCacheEntry
	flowCacheMux sync.Mutex

	prometheus v1.API

	permClient    permissions.Client
	tsClient      timescale_wrapper.Client
	servingClient *serving.Client

	ready bool
}

func NewController(ctx context.Context, conf configuration.Config, fatal func(err error)) (*Controller, error) {
	opencostClient, err := opencost.NewClient(conf)
	if err != nil {
		return nil, err
	}
	prometheusClient, err := api.NewClient(api.Config{
		Address: conf.PrometheusUrl,
	})
	if err != nil {
		return nil, err
	}

	permClient := permissions.NewClient(conf.PermissionsUrl)
	tsClient := timescale_wrapper.NewClient(conf.TimescaleWrapperUrl)
	servingClient := serving.New(conf.ServingUrl)

	controller := &Controller{opencost: opencostClient, config: conf, cache: map[string]cacheEntry{}, cacheMux: sync.Mutex{},
		parsingClient: parsing_api.NewParsingApi(conf.AnalyticsParsingUrl),
		operatorCache: map[string]operatorCacheEntry{}, operatorCacheMux: sync.Mutex{},
		flowCache: map[string]flowCacheEntry{}, flowCacheMux: sync.Mutex{},
		prometheus:    v1.NewAPI(prometheusClient),
		permClient:    permClient,
		tsClient:      tsClient,
		servingClient: servingClient,
		ready:         false,
	}

	if conf.Prefetch {
		go func() {
			controller.prefetch(fatal)
			controller.ready = true
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Minute):
					controller.prefetch(fatal)
				}
			}
		}()
	}
	return controller, nil
}

func (c *Controller) Ready() bool {
	return c.ready
}

func (c *Controller) prefetch(fatal func(err error)) {
	log.Println("Prefetching...")
	wg := sync.WaitGroup{}
	wg.Add(len(prefetchFn))
	for _, fn := range prefetchFn {
		fn := fn
		go func() {
			err := fn(c)
			if err != nil {
				fatal(err)
				return
			}
			wg.Done()
		}()
	}
	wg.Wait()
	log.Println("Prefetch done!")
}
