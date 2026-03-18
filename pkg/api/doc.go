/*
 *    Copyright 2020 InfAI (CC SES)
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
	"strings"

	_ "github.com/SENERGY-Platform/cost-calculator/docs"
	"github.com/SENERGY-Platform/cost-calculator/pkg/model"

	"github.com/SENERGY-Platform/cost-calculator/pkg/configuration"
	"github.com/SENERGY-Platform/cost-calculator/pkg/controller"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

func init() {
	endpoints = append(endpoints, DocEndpoint)
}

type ErrorResponse string

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -o ../../docs --parseDependency -d .. -g api/api.go
func DocEndpoint(router *gin.Engine, config configuration.Config, controller *controller.Controller) {
	router.GET("/doc", swaggerDocHandler)
}

// swaggerDocHandler godoc
// @Summary Get OpenAPI document
// @Description Returns the generated Swagger document for this service.
// @Tags documentation
// @Produce json
// @Success 200 {string} string
// @Failure 500 {string} ErrorResponse
// @Router /doc [get]
func swaggerDocHandler(c *gin.Context) {
	doc, err := swag.ReadDoc()
	if err != nil {
		_ = c.Error(errors.Join(err, model.ErrInternalServerError))
		return
	}
	// Remove empty host to let downstream tooling inject the correct target.
	doc = strings.Replace(doc, `"host": "",`, "", 1)
	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(doc))
}
