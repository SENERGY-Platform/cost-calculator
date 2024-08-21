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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SENERGY-Platform/opencost-wrapper/pkg/configuration"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/controller"
	"github.com/julienschmidt/httprouter"
)

func init() {
	endpoints = append(endpoints, CostsEndpoint)
}

func CostsEndpoint(router *httprouter.Router, config configuration.Config, controller *controller.Controller) {
	router.GET("/tree/:costType", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		userId, admin, err := getUserId(config, request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		token := getToken(request)
		overview, err := controller.GetCostControllers(userId, token, admin, params.ByName("costType"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(overview)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	})

	router.GET("/tree", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		userId, admin, err := getUserId(config, request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		token := getToken(request)
		overview, err := controller.GetCostTree(userId, token, admin)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(overview)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	})

	router.GET("/health", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		http.NoBody.WriteTo(writer)
	})
}
