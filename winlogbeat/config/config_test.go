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
		assert.Contains(t, v.config.Validate().Error(), v.errMsg)
	}
}

func TestConfigValidate(t *testing.T) {
	testCases := []validationTestCase{
		// Top-level config
		{
			WinlogbeatConfig{
				EventLogs: []EventLogConfig{
					{Name: "App"},
				},
			},
			"", // No Error
		},
		{
			Settings{
				WinlogbeatConfig{
					EventLogs: []EventLogConfig{
						{Name: "App"},
					},
				},
				map[string]interface{}{"other": "value"},
			},
			"1 error: Invalid top-level key 'other' found. Valid keys are " +
				"logging, output, shipper, winlogbeat",
		},
		{
			WinlogbeatConfig{},
			"1 error: At least one event log must be configured as part of " +
				"event_logs",
		},
		{
			WinlogbeatConfig{IgnoreOlder: "1"},
			"2 errors: Invalid top level ignore_older value '1' (time: " +
				"missing unit in duration 1); At least one event log must be " +
				"configured as part of event_logs",
		},
		{
			WinlogbeatConfig{
				EventLogs: []EventLogConfig{
					{Name: "App"},
				},
				Metrics: MetricsConfig{BindAddress: "example.com"},
			},
			"1 error: bind_address",
		},
		{
			WinlogbeatConfig{
				EventLogs: []EventLogConfig{
					{},
				},
			},
			"1 error: event log is missing a 'name'",
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
		// EventLogConfig
		{
			EventLogConfig{Name: "System"},
			"",
		},
		{
			EventLogConfig{},
			"event log is missing a 'name'",
		},
		{
			EventLogConfig{Name: "System", IgnoreOlder: "24"},
			"Invalid ignore_older value ('24') for event_log 'System' " +
				"(time: missing unit in duration 24)",
		},
		{
			EventLogConfig{Name: "System", API: "eventlogging"},
			"",
		},
		{
			EventLogConfig{Name: "System", API: "wineventlog"},
			"",
		},
		{
			EventLogConfig{Name: "System", API: "invalid"},
			"Invalid api value ('invalid') for event_log 'System'",
		},
	}

	for _, test := range testCases {
		test.run(t)
	}
}
