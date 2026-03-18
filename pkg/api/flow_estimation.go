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
	router.GET("/estimation/flow/:id", getFlowEstimationHandler(config, controller))
	router.POST("/estimation/flow", postFlowEstimationHandler(config, controller))
}

// getFlowEstimationHandler godoc
// @Summary Get flow cost estimation
// @Description Returns the cost estimation for a single flow ID.
// @Tags estimations
// @Produce json
// @Param id path string true "Flow ID"
// @Success 200 {array} model.Estimation
// @Failure 400 {string} ErrorResponse
// @Failure 500 {string} ErrorResponse
// @Router /estimation/flow/{id} [get]
func getFlowEstimationHandler(config configuration.Config, controller *controller.Controller) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// postFlowEstimationHandler godoc
// @Summary Get flow cost estimations
// @Description Returns cost estimations for the provided list of flow IDs.
// @Tags estimations
// @Accept json
// @Produce json
// @Param ids body []string true "Flow IDs"
// @Success 200 {array} model.Estimation
// @Failure 400 {string} ErrorResponse
// @Failure 500 {string} ErrorResponse
// @Router /estimation/flow [post]
func postFlowEstimationHandler(config configuration.Config, controller *controller.Controller) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
