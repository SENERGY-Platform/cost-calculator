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
	permissions "github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var prefetchFn = []func(c *Controller) error{}

type operatorCacheEntry struct {
	estimation model.Estimation
	enteredAt  time.Time
}

type flowCacheEntry struct {
	estimation *model.Estimation
	enteredAt  time.Time
}

type Controller struct {
	config        configuration.Config
	parsingClient *parsing_api.ParsingApi
	flowCache     map[string]flowCacheEntry
	flowCacheMux  sync.Mutex
	prometheus    v1.API

	permClient    permissions.Client
	servingClient *serving.Client

	pricingModel *model.PricingModel
}

func NewController(ctx context.Context, conf configuration.Config, fatal func(err error)) (*Controller, error) {
	pricingModel, err := model.GetPricingModel(conf.PricingModelFilePath)
	prometheusClient, err := api.NewClient(api.Config{
		Address: conf.PrometheusUrl,
	})
	if err != nil {
		return nil, err
	}

	permClient := permissions.NewClient(conf.PermissionsUrl)
	servingClient := serving.New(conf.ServingUrl)

	controller := &Controller{config: conf,
		parsingClient: parsing_api.NewParsingApi(conf.AnalyticsParsingUrl),
		prometheus:    v1.NewAPI(prometheusClient),
		permClient:    permClient,
		servingClient: servingClient,
		pricingModel:  &pricingModel,
		flowCache:     map[string]flowCacheEntry{}, flowCacheMux: sync.Mutex{},
	}

	return controller, nil
}

func (c *Controller) logDebug(s string) {
	if c.config.Debug {
		log.Println("DEBUG: " + s)
	}
}
