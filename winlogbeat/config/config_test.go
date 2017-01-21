// +build !integration

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type validationTestCase struct {
	config Validator
	errMsg string
}

func (v validationTestCase) run(t *testing.T) {
	if v.errMsg == "" {
		assert.NoError(t, v.config.Validate())
	} else {
		err := v.config.Validate()
		if assert.Error(t, err, "expected '%s'", v.errMsg) {
			assert.Contains(t, err.Error(), v.errMsg)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []validationTestCase{
		// Top-level config
		{
			WinlogbeatConfig{
				EventLogs: []map[string]interface{}{
					{"Name": "App"},
				},
			},
			"", // No Error
		},
		{
			Settings{
				WinlogbeatConfig{
					EventLogs: []map[string]interface{}{
						{"Name": "App"},
					},
				},
				map[string]interface{}{"other": "value"},
			},
			"1 error: Invalid top-level key 'other' found. Valid keys are bulk_queue_size, " +
				"fields, fields_under_root, geoip, logging, max_procs, " +
				"name, output, path, processors, queue_size, refresh_topology_freq, tags, topology_expire, winlogbeat",
		},
		{
			WinlogbeatConfig{},
			"1 error: At least one event log must be configured as part of " +
				"event_logs",
		},
		{
			WinlogbeatConfig{
				EventLogs: []map[string]interface{}{
					{"Name": "App"},
				},
				Metrics: MetricsConfig{BindAddress: "example.com"},
			},
			"1 error: bind_address",
		},
		// MetricsConfig
		{
			MetricsConfig{},
			"",
		},
		{
			MetricsConfig{BindAddress: "example.com:6700"},
			"",
		},
		{
			MetricsConfig{BindAddress: "example.com"},
			"bind_address must be formatted as host:port but was " +
				"'example.com' (missing port in address example.com)",
		},
		{
			MetricsConfig{BindAddress: ":1"},
			"bind_address value (':1') is missing a host",
		},
		{
			MetricsConfig{BindAddress: "example.com:1024f"},
			"bind_address port value ('1024f') must be a number " +
				"(strconv.ParseInt: parsing \"1024f\": invalid syntax)",
		},
		{
			MetricsConfig{BindAddress: "example.com:0"},
			"bind_address port must be within [1-65535] but was '0'",
		},
		{
			MetricsConfig{BindAddress: "example.com:65536"},
			"bind_address port must be within [1-65535] but was '65536'",
		},
	}

	for _, test := range testCases {
		test.run(t)
	}
}
