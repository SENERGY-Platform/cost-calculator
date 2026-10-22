/*
 *    Copyright (c) 2026 InfAI (CC SES)
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

package configuration

import (
	"slices"
	"testing"
)

// A map with lists as values comes from the environment as JSON; a flat map keeps the form
// key:value,key:value.
func TestMapsFromTheEnvironment(t *testing.T) {
	t.Setenv("PROCESS_COST_SOURCES", `{"automation-engine": ["engine.*", "events.*"], "process-task-worker": ["io-worker"]}`)
	t.Setenv("PROCESS_COST_SOURCE_TO_INSTANCE_ID_PLACEHOLDER_FOR_PROCESS_DEF_COST_FRACTION", "engine: internal, events: events-internal")
	config, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(config.ProcessCostSources) != 2 || !slices.Equal(config.ProcessCostSources["automation-engine"], []string{"engine.*", "events.*"}) {
		t.Errorf("process cost sources: %+v", config.ProcessCostSources)
	}
	if got := config.ProcessCostSourceToInstanceIdPlaceholderForProcessDefCostFraction; len(got) != 2 || got["engine"] != "internal" {
		t.Errorf("instance ids: %+v", got)
	}
}
