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

package jetstream

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMappingFor(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/all.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	// Enable all data points to ensure each individual mapper
	// does not result in error.
	config := ModuleConfig{
		Jetstream: MetricsetConfig{
			Stats: StatsConfig{
				Enabled: true,
			},
			Account: AccountConfig{
				Enabled: true,
			},
			Stream: StreamConfig{
				Enabled: true,
			},
			Consumer: ConsumerConfig{
				Enabled: true,
			},
		},
	}
	ms := &MetricSet{
		Config: config.Jetstream,
	}
	err = eventMapping(ms, reporter, content)
	assert.NoError(t, err)
}

func TestFetchEventContentForStats(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:                      "http",
		URL:                       "/jsz?config=true",
		Suffix:                    "json",
		Path:                      "_meta/test/stats",
		WritePath:                 "_meta/testdata/stats",
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.*"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"stats": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}

func TestFetchEventContentForAccount(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:                      "http",
		URL:                       "/jsz?accounts=true&config=true",
		Suffix:                    "json",
		Path:                      "_meta/test/accounts",
		WritePath:                 "_meta/testdata/accounts",
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.*"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"account": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}
