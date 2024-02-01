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

package opencost

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

func (c *Client) Allocation(options *AllocationOptions) (AllocationResponse, error) {
	query := options.toQuery()
	url := c.config.OpencostUrl + "/allocation" + query
	cacheE, ok := c.cache[url]
	if ok && time.Now().Before(cacheE.validUntil) {
		return cacheE.value.(AllocationResponse), nil
	}
	var allo AllocationResponse

	log.Println("query opencost:", query)
	start := time.Now()
	defer func() {
		log.Printf("query opencost done: %v in %v", query, time.Since(start).String())
	}()
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return allo, err
	}
	if resp.StatusCode != http.StatusOK {
		return allo, errors.New("unexpected upstreams statuscode")
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&allo)
	c.cacheMux.Lock()
	defer c.cacheMux.Unlock()
	c.cache[url] = cacheEntry{
		value:      allo,
		validUntil: time.Now().Add(15 * time.Minute),
	}
	return allo, err
}
