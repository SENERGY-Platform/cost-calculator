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

package pkg

import (
	"context"

	"github.com/SENERGY-Platform/cost-calculator/pkg/api"
	"github.com/SENERGY-Platform/cost-calculator/pkg/configuration"
	"github.com/SENERGY-Platform/cost-calculator/pkg/controller"

	"sync"
)

func Start(ctx context.Context, config configuration.Config, fatal func(err error)) (wg *sync.WaitGroup, err error) {
	wg = &sync.WaitGroup{}
	ctrl, err := controller.NewController(ctx, config, fatal)
	if err != nil {
		return wg, err
	}

	err = api.Start(ctx, wg, config, ctrl)
	return
}
