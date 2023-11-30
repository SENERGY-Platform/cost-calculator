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
	"time"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/configuration"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
)

var prefetchFn = []func(c *Controller) error{}

type cacheEntry struct {
	allocation *opencost.AllocationResponse
	enteredAt  time.Time
}

type Controller struct {
	opencost *opencost.Client
	config   configuration.Config
	cache    map[string]cacheEntry
}

func NewController(ctx context.Context, conf configuration.Config, fatal func(err error)) (*Controller, error) {
	opencostClient, err := opencost.NewClient(conf)
	controller := &Controller{opencost: opencostClient, config: conf, cache: map[string]cacheEntry{}}
	if err != nil {
		return nil, err
	}
	if conf.Prefetch {
		prefetch := func() {
			log.Println("Prefetching...")
			for _, fn := range prefetchFn {
				if ctx.Err() != nil {
					return
				}
				err := fn(controller)
				if err != nil {
					fatal(err)
					return
				}
			}
			log.Println("Prefetch done!")
		}
		go func() {
			prefetch()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Minute):
					prefetch()
				}
			}
		}()
	}
	return controller, nil
}
