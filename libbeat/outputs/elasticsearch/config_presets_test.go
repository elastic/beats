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

package elasticsearch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestApplyPresetNoConflicts(t *testing.T) {
	const testHost = "http://elastic-host:9200"
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"hosts": []string{testHost},

		// Set some parameters that aren't affected by performance presets
		"max_retries": 5,
		"loadbalance": true,
	})
	// Apply the preset and make sure no conflicts are reported.
	conflicts, _, err := applyPreset(presetThroughput, cfg)
	require.NoError(t, err, "Valid preset must apply successfully")
	assert.Equal(t, 0, len(conflicts), "applyPreset should report no conflicts from non-preset fields")

	// Unpack the final config into elasticsearchConfig and verify that both user
	// and preset fields are set correctly.
	esConfig := elasticsearchConfig{}
	err = cfg.Unpack(&esConfig)
	require.NoError(t, err, "Config should unpack successfully")

	// Check basic user params
	assert.Equal(t, 5, esConfig.MaxRetries, "Non-preset fields should be unchanged by applyPreset")
	assert.Equal(t, true, esConfig.LoadBalance, "Non-preset fields should be unchanged by applyPreset")

	// Check basic preset params
	assert.Equal(t, 1600, esConfig.BulkMaxSize, "Preset fields should be set by applyPreset")
	assert.Equal(t, 1, esConfig.CompressionLevel, "Preset fields should be set by applyPreset")
	assert.Equal(t, 15*time.Second, esConfig.Transport.IdleConnTimeout, "Preset fields should be set by applyPreset")

	// Check preset queue params
	var memQueueConfig struct {
		Events         int           `config:"events"`
		FlushMinEvents int           `config:"flush.min_events"`
		FlushTimeout   time.Duration `config:"flush.timeout"`
	}
	require.Equal(t, "mem", esConfig.Queue.Name(), "applyPreset should configure the memory queue")
	err = esConfig.Queue.Config().Unpack(&memQueueConfig)
	assert.NoError(t, err, "applyPreset should set valid memory queue config")

	assert.Equal(t, 12800, memQueueConfig.Events, "Queue fields should match preset definition")
	assert.Equal(t, 1600, memQueueConfig.FlushMinEvents, "Queue fields should match preset definition")
	assert.Equal(t, 5*time.Second, memQueueConfig.FlushTimeout, "Queue fields should match preset definition")

	// Check calculated hosts, which should contain one copy of the user config
	// hosts for each configured worker (which for presetThroughput is 4).
	hosts, err := outputs.ReadHostList(cfg)
	require.NoError(t, err, "ReadHostList should succeed")
	assert.Equal(t, 4, len(hosts), "'throughput' preset should create 4 workers per host")
	for _, host := range hosts {
		assert.Equal(t, testHost, host, "Computed hosts should match user config")
	}
}

func TestApplyPresetWithConflicts(t *testing.T) {
	const testHost = "http://elastic-host:9200"
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"hosts": []string{testHost},

		// Set parameters contained in the performance presets, with
		// arbitrary numbers that do not match the preset values so we can
		// make sure everything is overridden.
		"bulk_max_size":              100,
		"worker":                     10,
		"queue.mem.events":           1000,
		"queue.mem.flush.min_events": 100,
		"queue.mem.flush.timeout":    100 * time.Second,
		"compression_level":          5,
		"idle_connection_timeout":    100 * time.Second,
	})
	// Apply the preset and ensure all preset fields are reported as conflicts
	conflicts, _, err := applyPreset(presetBalanced, cfg)
	require.NoError(t, err, "Valid preset must apply successfully")
	expectedConflicts := []string{
		"bulk_max_size",
		"worker",
		"queue.mem.events",
		"queue.mem.flush.min_events",
		"queue.mem.flush.timeout",
		"compression_level",
		"idle_connection_timeout",
	}
	assert.ElementsMatch(t, expectedConflicts, conflicts, "All preset fields should be reported as overridden")

	// Unpack the final config into elasticsearchConfig and verify that user
	// fields were overridden
	esConfig := elasticsearchConfig{}
	err = cfg.Unpack(&esConfig)
	require.NoError(t, err, "Valid config tree must unpack successfully")

	// Check basic preset params
	assert.Equal(t, 1600, esConfig.BulkMaxSize, "Preset fields should be set by applyPreset")
	assert.Equal(t, 1, esConfig.CompressionLevel, "Preset fields should be set by applyPreset")
	assert.Equal(t, 3*time.Second, esConfig.Transport.IdleConnTimeout, "Preset fields should be set by applyPreset")

	// Check preset queue params
	var memQueueConfig struct {
		Events         int           `config:"events"`
		FlushMinEvents int           `config:"flush.min_events"`
		FlushTimeout   time.Duration `config:"flush.timeout"`
	}
	require.Equal(t, "mem", esConfig.Queue.Name(), "applyPreset should configure the memory queue")
	err = esConfig.Queue.Config().Unpack(&memQueueConfig)
	assert.NoError(t, err, "applyPreset should set valid memory queue config")

	assert.Equal(t, 3200, memQueueConfig.Events, "Queue fields should match preset definition")
	assert.Equal(t, 1600, memQueueConfig.FlushMinEvents, "Queue fields should match preset definition")
	assert.Equal(t, 10*time.Second, memQueueConfig.FlushTimeout, "Queue fields should match preset definition")

	// Check calculated hosts, which should contain one copy of the user config
	// hosts for each configured worker (which for presetBalanced is 1).
	hosts, err := outputs.ReadHostList(cfg)
	require.NoError(t, err, "ReadHostList should succeed")
	require.Equal(t, 1, len(hosts), "'balanced' preset should create 1 worker per host")
	assert.Equal(t, testHost, hosts[0])
}

func TestApplyPresetCustom(t *testing.T) {
	const testHost = "http://elastic-host:9200"
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"hosts": []string{testHost},

		// Set parameters contained in the performance presets, with
		// arbitrary numbers that do not match the preset values so we can
		// make sure nothing is overridden.
		"bulk_max_size":              100,
		"worker":                     2,
		"queue.mem.events":           1000,
		"queue.mem.flush.min_events": 100,
		"queue.mem.flush.timeout":    100 * time.Second,
		"compression_level":          5,
		"idle_connection_timeout":    100 * time.Second,
	})
	// Apply the preset and make sure no conflicts are reported.
	conflicts, _, err := applyPreset(presetCustom, cfg)
	require.NoError(t, err, "Custom preset must apply successfully")
	assert.Equal(t, 0, len(conflicts), "applyPreset should report no conflicts when preset is 'custom'")

	// Unpack the final config into elasticsearchConfig and verify that both user
	// and preset fields are set correctly.
	esConfig := elasticsearchConfig{}
	err = cfg.Unpack(&esConfig)
	require.NoError(t, err, "Config should unpack successfully")

	// Check basic user params
	assert.Equal(t, 100, esConfig.BulkMaxSize, "Preset fields should be set by applyPreset")
	assert.Equal(t, 5, esConfig.CompressionLevel, "Preset fields should be set by applyPreset")
	assert.Equal(t, 100*time.Second, esConfig.Transport.IdleConnTimeout, "Preset fields should be set by applyPreset")

	// Check user queue params
	var memQueueConfig struct {
		Events         int           `config:"events"`
		FlushMinEvents int           `config:"flush.min_events"`
		FlushTimeout   time.Duration `config:"flush.timeout"`
	}
	require.Equal(t, "mem", esConfig.Queue.Name(), "applyPreset with custom preset should preserve user queue settings")
	err = esConfig.Queue.Config().Unpack(&memQueueConfig)
	assert.NoError(t, err, "Queue settings should unpack successfully")

	assert.Equal(t, 1000, memQueueConfig.Events, "Queue fields should match preset definition")
	assert.Equal(t, 100, memQueueConfig.FlushMinEvents, "Queue fields should match preset definition")
	assert.Equal(t, 100*time.Second, memQueueConfig.FlushTimeout, "Queue fields should match preset definition")

	// Check calculated hosts, which should contain one copy of the user config
	// hosts for each configured worker (which in this case means 2).
	hosts, err := outputs.ReadHostList(cfg)
	require.NoError(t, err, "ReadHostList should succeed")
	assert.Equal(t, 2, len(hosts), "'custom' preset should leave worker count unchanged")
	for _, host := range hosts {
		assert.Equal(t, testHost, host, "Computed hosts should match user config")
	}
}

func TestFlattenedKeysRemovesNamespace(t *testing.T) {
	// A test exhibiting the namespace corner case that breaks the baseline
	// behavior of FlattenedKeys, and ensuring that flattenedKeysForConfig
	// fixes it.
	rawCfg := config.MustNewConfigFrom(map[string]interface{}{
		"namespace.testkey": "testvalue",
	})
	ns := config.Namespace{}
	err := ns.Unpack(rawCfg)
	require.NoError(t, err, "Namespace unpack should succeed")

	// Extract the sub-config from the Namespace object.
	cfg := ns.Config()

	// FlattenedKeys on the config object parsed via a Namespace will still
	// include the namespace in the reported keys
	nsFlattenedKeys := cfg.FlattenedKeys()
	assert.ElementsMatch(t, []string{"namespace.testkey"}, nsFlattenedKeys,
		"Expected keys from FlattenedKeys to include original namespace")

	// flattenedKeysForConfig should strip the namespace prefix so we can
	// reliably compare output config fields no matter how they were
	// originally created.
	cfgFlattenedKeys, err := flattenedKeysForConfig(cfg)
	require.NoError(t, err, "flattenedKeysForConfig should succeed")
	assert.ElementsMatch(t, []string{"testkey"}, cfgFlattenedKeys,
		"Expected flattenedKeysForConfig to remove original namespace")
}
