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

package api

import (
	"errors"
	"net/http"

	"github.com/SENERGY-Platform/cost-calculator/pkg/configuration"
	"github.com/SENERGY-Platform/cost-calculator/pkg/controller"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	"github.com/gin-gonic/gin"
)

func init() {
	endpoints = append(endpoints, FlowEstimationEndpoint)
}

func FlowEstimationEndpoint(router *gin.Engine, config configuration.Config, controller *controller.Controller) {
	router.GET("/estimation/flow/:id", func(c *gin.Context) {
		userId, _, err := getUserId(config, c.Request)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		token := getToken(c.Request)
		overview, err := controller.GetFlowEstimations(token, userId, []string{c.Param("id")})
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}
		c.JSON(http.StatusOK, overview)
	})

	router.POST("/estimation/flow", func(c *gin.Context) {
		userId, _, err := getUserId(config, c.Request)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		token := getToken(c.Request)
		flowsIds := []string{}
		err = c.ShouldBind(&flowsIds)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}

		result, err := controller.GetFlowEstimations(token, userId, flowsIds)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}

		c.JSON(http.StatusOK, result)
	})
}
