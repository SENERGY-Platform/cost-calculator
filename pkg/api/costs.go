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
	"net/url"
	"strconv"
	"time"

	"github.com/SENERGY-Platform/cost-calculator/pkg/configuration"
	"github.com/SENERGY-Platform/cost-calculator/pkg/controller"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"
	"github.com/gin-gonic/gin"
)

func init() {
	endpoints = append(endpoints, CostsEndpoint)
}

func CostsEndpoint(router *gin.Engine, config configuration.Config, controller *controller.Controller) {
	router.GET("/tree/:costType", getCostControllersHandler(config, controller))
	router.GET("/tree", getCostTreeHandler(config, controller))
	router.GET("/health", healthHandler)
}

// getCostControllersHandler godoc
// @Summary Get cost tree for a single cost type
// @Description Returns the detailed cost tree for one cost type for the current user or a delegated user.
// @Tags costs
// @Produce json
// @Param costType path string true "Cost type" Enums(analytics, imports, API Calls, Exports, Devices, process, MQTTExports, Reporting)
// @Param skip_estimation query bool false "Skip estimation values in the response"
// @Param start query string false "Start time in RFC3339 format"
// @Param end query string false "End time in RFC3339 format"
// @Param for_user query string false "User ID to query as admin"
// @Success 200 {object} model.CostWithChildren
// @Failure 400 {string} ErrorResponse
// @Failure 500 {string} ErrorResponse
// @Router /tree/{costType} [get]
func getCostControllersHandler(config configuration.Config, controller *controller.Controller) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, admin, err := getUserId(config, c.Request)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		token := getToken(c.Request)
		skipEstimation := false
		if len(c.Query("skip_estimation")) > 0 {
			skipEstimation, err = strconv.ParseBool(c.Query("skip_estimation"))
			if err != nil {
				_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
				return
			}
		}
		start, end, err := parseStartEnd(c.Request.URL.Query())
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}

		overview, err := controller.GetCostControllers(userId, token, admin, c.Param("costType"), skipEstimation, start, end)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}
		c.JSON(http.StatusOK, overview)
	}
}

// getCostTreeHandler godoc
// @Summary Get aggregated cost tree
// @Description Returns the aggregated cost tree across all supported cost types for the current user or a delegated user.
// @Tags costs
// @Produce json
// @Param skip_estimation query bool false "Skip estimation values in the response"
// @Param start query string false "Start time in RFC3339 format"
// @Param end query string false "End time in RFC3339 format"
// @Param for_user query string false "User ID to query as admin"
// @Success 200 {object} model.CostTree
// @Failure 400 {string} ErrorResponse
// @Failure 500 {string} ErrorResponse
// @Router /tree [get]
func getCostTreeHandler(config configuration.Config, controller *controller.Controller) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, admin, err := getUserId(config, c.Request)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		token := getToken(c.Request)
		skipEstimation := false
		if len(c.Query("skip_estimation")) > 0 {
			skipEstimation, err = strconv.ParseBool(c.Query("skip_estimation"))
			if err != nil {
				_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
				return
			}
		}
		start, end, err := parseStartEnd(c.Request.URL.Query())
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		overview, err := controller.GetCostTree(userId, token, admin, skipEstimation, start, end)
		if err != nil {
			_ = c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}
		c.JSON(http.StatusOK, overview)
	}
}

// healthHandler godoc
// @Summary Health check
// @Description Returns a successful status when the API is ready to serve requests.
// @Tags health
// @Success 200
// @Router /health [get]
func healthHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

func parseStartEnd(values url.Values) (start, end *time.Time, err error) {
	if len(values.Get("start")) > 0 {
		s, err := time.Parse(time.RFC3339, values.Get("start"))
		if err != nil {
			return start, end, err
		}
		start = &s
	}
	if len(values.Get("end")) > 0 {
		s, err := time.Parse(time.RFC3339, values.Get("end"))
		if err != nil {
			return start, end, err
		}
		end = &s
	}
	return
}
