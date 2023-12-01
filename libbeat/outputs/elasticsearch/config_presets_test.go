package elasticsearch

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	noPresetCfg := config.MustNewConfigFrom(map[string]interface{}{})
	validPresetCfg := config.MustNewConfigFrom(map[string]interface{}{
		"preset": "balanced",
	})
	invalidPresetCfg := config.MustNewConfigFrom(map[string]interface{}{
		"preset": "asdf",
	})

	cfg := elasticsearchConfig{}
	err := noPresetCfg.Unpack(&cfg)
	require.NoError(t, err, "Config with no specified preset should unpack successfully")

	err = validPresetCfg.Unpack(&cfg)
	require.NoError(t, err, "Config with valid preset should unpack successfully")

	err = invalidPresetCfg.Unpack(&cfg)
	require.Error(t, err, "Config with invalid preset should not unpack successfully")
}

func TestApplyPresetNoConflicts(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"preset": presetThroughput,
		// Set some parameters that aren't affected by performance presets
		"max_retries": 5,
		"username":    "elastic",
		"password":    "password",
		"loadbalance": true,
	})
	esConfig := elasticsearchConfig{}
	err := cfg.Unpack(&esConfig)
	require.NoError(t, err, "Valid config tree must unpack successfully")

	// Apply the preset and make sure:
	// - the resulting config contains the original and preset values
	// - no conflicts are reported
	conflicts, err := applyPreset(&esConfig, cfg)
	require.NoError(t, err, "Valid preset must apply successfully")
	assert.Equal(t, 0, len(conflicts), "applyPreset should report no conflicts from non-preset fields")

	assert.Equal(t, 5, esConfig.MaxRetries, "Non-preset fields should be unchanged by applyPreset")
	assert.Equal(t, "elastic", esConfig.Username, "Non-preset fields should be unchanged by applyPreset")
	assert.Equal(t, "password", esConfig.Password, "Non-preset fields should be unchanged by applyPreset")
	assert.Equal(t, true, esConfig.LoadBalance, "Non-preset fields should be unchanged by applyPreset")

	assert.Equal(t, 1600, esConfig.BulkMaxSize, "Preset fields should be set by applyPreset")
	assert.Equal(t, 1, esConfig.CompressionLevel, "Preset fields should be set by applyPreset")
	assert.Equal(t, 15*time.Second, esConfig.Transport.IdleConnTimeout, "Preset fields should be set by applyPreset")
	// TODO: rework applyPreset to operate directly on config structs before
	// the first unpack, so we can properly set (and test) the worker count.
	//assert.Equal(t, 4, esConfig.workers, "Preset fields should be set by applyPreset")

	// Queue params are more awkward to test
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
}

func TestApplyPresetWithConflicts(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"preset": presetThroughput,
		// Set parameters contained in the performance presets, with
		// arbitrary numbers that do not match the preset values so we can
		// make sure everything is overridden.
		"bulk_max_size":              100,
		"workers":                    10,
		"queue.mem.events":           1000,
		"queue.mem.flush.min_events": 100,
		"queue.mem.flush.timeout":    100 * time.Second,
		"compression_level":          5,
		"idle_connection_timeout":    100 * time.Second,
	})
	esConfig := elasticsearchConfig{}
	err := cfg.Unpack(&esConfig)
	require.NoError(t, err, "Valid config tree must unpack successfully")

	// Apply the preset and make sure:
	// - the resulting config overrides all initial config values
	// - conflicts are reported on all fields
	conflicts, err := applyPreset(&esConfig, cfg)
	require.NoError(t, err, "Valid preset must apply successfully")
	assert.Equal(t, 7, len(conflicts), "Number of conflicts should equal number of overridden fields")
	// TODO: add the remaining equality checks
}

func TestApplyPresetCustom(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"preset": presetCustom,
		// Set parameters contained in the performance presets, with
		// arbitrary numbers that do not match the preset values so we can
		// make sure nothing is overridden.
		"bulk_max_size":              100,
		"workers":                    10,
		"queue.mem.events":           1000,
		"queue.mem.flush.min_events": 100,
		"queue.mem.flush.timeout":    100 * time.Second,
		"compression_level":          5,
		"idle_connection_timeout":    100 * time.Second,
	})
	esConfig := elasticsearchConfig{}
	err := cfg.Unpack(&esConfig)
	require.NoError(t, err, "Valid config tree must unpack successfully")

	// Apply the preset and make sure:
	// - the resulting config overrides all initial config values
	// - conflicts are reported on all fields
	conflicts, err := applyPreset(&esConfig, cfg)
	require.NoError(t, err, "Custom preset must apply successfully")
	assert.Equal(t, 0, len(conflicts), "Custom preset should always report no conflicts")
	// TODO: add the remaining equality checks
}
