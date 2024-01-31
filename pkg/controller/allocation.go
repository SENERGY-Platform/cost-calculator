/*
 *    Copyright 2024 InfAI (CC SES)
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
	"errors"
	"github.com/SENERGY-Platform/opencost-wrapper/pkg/opencost"
	"log"
	"time"
)

func optionToKey(options opencost.AllocationOptions) string {
	return options.Window + "|" + options.Aggregate
}

func (this *Controller) GetCachedAllocation(options opencost.AllocationOptions) (allocation opencost.AllocationResponse, err error) {
	key := optionToKey(options)

	this.cacheMux.Lock()
	cached, ok := this.cache[key]
	this.cacheMux.Unlock()
	if ok && cached.enteredAt.Add(cacheValid).After(time.Now()) {
		allocation = cached.allocation
	} else if this.config.Prefetch {
		return allocation, errors.New("prefetch enabled, but cache empty or outdated, try again later")
	} else {
		allocation, err = this.GetAllocation(&options)
		if err != nil {
			return allocation, err
		}
	}
	return allocation, nil
}

func (this *Controller) GetAllocation(options *opencost.AllocationOptions) (allocation opencost.AllocationResponse, err error) {
	allocation, err = this.opencost.Allocation(options)
	if err != nil {
		return allocation, err
	}
	err = validateAllocation(&allocation)
	if err != nil {
		return allocation, err
	}

	return allocation, nil
}

func GetPrefetchAllocationFunction(options opencost.AllocationOptions) func(c *Controller) error {
	return func(c *Controller) error {
		key := optionToKey(options)
		allocation, err := c.opencost.Allocation(&options)
		if err != nil {
			log.Println("WARNING: Could not prefetch: " + err.Error())
			return err
		}
		err = validateAllocation(&allocation)
		if err != nil {
			log.Println("WARNING: Could not prefetch, invalid allocation response: " + err.Error())
			return err
		}
		c.cacheMux.Lock()
		defer c.cacheMux.Unlock()
		c.cache[key] = cacheEntry{
			allocation: allocation,
			enteredAt:  time.Now(),
		}
		return nil
	}
}
