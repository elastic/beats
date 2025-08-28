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

func TestEventMappingForErrors(t *testing.T) {
	content, err := os.ReadFile("./_meta/testdata/full-results.json")
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

// This is a basic happy-path test for only reporting on stats
func TestFetchEventContentForStats(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "stats.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.category"},
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

// This is a basic happy-path test for only reporting on accounts
func TestFetchEventContentForAccount(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "accounts.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.account.accounts"},
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

// This is a basic happy-path test for only reporting on streams
func TestFetchEventContentForStreams(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "streams.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.category"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"stream": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}

// This is a basic happy-path test for only reporting on consumers
func TestFetchEventContentForConsumers(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "consumers.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.category"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"consumer": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}

// This is a basic happy-path test for reporting on all data points
func TestFetchEventContentForAll(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "all.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.category", "nats.jetstream.account.accounts"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"stats": map[string]interface{}{
					"enabled": true,
				},
				"account": map[string]interface{}{
					"enabled": true,
				},
				"stream": map[string]interface{}{
					"enabled": true,
				},
				"consumer": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}

// This is a basic happy-path test for reporting on all data points
func TestFetchEventContentForAllWithNothingEnabled(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "all.disabled.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"stats": map[string]interface{}{
					"enabled": false,
				},
				"account": map[string]interface{}{
					"enabled": false,
				},
				"stream": map[string]interface{}{
					"enabled": false,
				},
				"consumer": map[string]interface{}{
					"enabled": false,
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}

// This is a basic happy-path test for testing filters
func TestFetchEventContentForAllWithFilters(t *testing.T) {
	dataConfig := mbtest.DataConfig{
		Type:      "http",
		URL:       "/jsz?config=true&consumers=true",
		Suffix:    "all.filters.json",
		Path:      "_meta/testdata/input",
		WritePath: "_meta/testdata/expected",
		// Not sure why this field isn't being recognized as documented. It does exist.
		OmitDocumentedFieldsCheck: []string{"nats.jetstream.category", "nats.jetstream.account.accounts"},
		Module: map[string]interface{}{
			"jetstream": map[string]interface{}{
				"stats": map[string]interface{}{
					"enabled": true,
				},
				"account": map[string]interface{}{
					"enabled": true,
					"names":   []string{"account-2"},
				},
				"stream": map[string]interface{}{
					"enabled": true,
					"names":   []string{"test-stream-2"},
				},
				"consumer": map[string]interface{}{
					"enabled": true,
					"names":   []string{"test-stream-2-consumer-2"},
				},
			},
		},
	}
	mbtest.TestDataFilesWithConfig(t, "nats", "jetstream", dataConfig, "")
}
