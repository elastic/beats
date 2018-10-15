// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build integration

package queue

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "rabbitmq")

	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEventsCond(f, t, func(e common.MapStr) bool {
		hasTotal, _ := e.HasKey("messages.total")
		return hasTotal
	})
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	config := mtest.GetIntegrationConfig()
	config["metricsets"] = []string{"queue"}
	return config
}
