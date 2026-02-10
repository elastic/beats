// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsqueryConfig_Render_ScheduleSplayPercent(t *testing.T) {
	tests := []struct {
		name           string
		config         OsqueryConfig
		expectedSplay  interface{}
		expectedInJSON bool
	}{
		{
			name:           "nil splay percent - not included in output",
			config:         OsqueryConfig{},
			expectedInJSON: false,
		},
		{
			name: "splay percent set to 10",
			config: OsqueryConfig{
				ScheduleSplayPercent: intPtr(10),
			},
			expectedSplay:  float64(10), // JSON unmarshals numbers as float64
			expectedInJSON: true,
		},
		{
			name: "splay percent set to 0 - disables splay",
			config: OsqueryConfig{
				ScheduleSplayPercent: intPtr(0),
			},
			expectedSplay:  float64(0),
			expectedInJSON: true,
		},
		{
			name: "splay percent does not override explicit options value",
			config: OsqueryConfig{
				Options: map[string]interface{}{
					"schedule_splay_percent": 20,
				},
				ScheduleSplayPercent: intPtr(10),
			},
			expectedSplay:  float64(20), // The explicit options value should be preserved
			expectedInJSON: true,
		},
		{
			name: "splay percent with existing options map",
			config: OsqueryConfig{
				Options: map[string]interface{}{
					"schedule_timeout": 300,
				},
				ScheduleSplayPercent: intPtr(15),
			},
			expectedSplay:  float64(15),
			expectedInJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := tt.config.Render()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(rendered, &result)
			require.NoError(t, err)

			if tt.expectedInJSON {
				options, ok := result["options"].(map[string]interface{})
				require.True(t, ok, "options should be present in rendered output")
				assert.Equal(t, tt.expectedSplay, options["schedule_splay_percent"])
			} else {
				// Either no options or no schedule_splay_percent in options
				if options, ok := result["options"].(map[string]interface{}); ok {
					_, hasSplay := options["schedule_splay_percent"]
					assert.False(t, hasSplay, "schedule_splay_percent should not be in options")
				}
			}
		})
	}
}

func TestOsqueryConfig_Render_ScheduleMaxDrift(t *testing.T) {
	tests := []struct {
		name           string
		config         OsqueryConfig
		expectedDrift  interface{}
		expectedInJSON bool
	}{
		{
			name:           "nil max drift - not included in output",
			config:         OsqueryConfig{},
			expectedInJSON: false,
		},
		{
			name: "max drift set to 60",
			config: OsqueryConfig{
				ScheduleMaxDrift: intPtr(60),
			},
			expectedDrift:  float64(60),
			expectedInJSON: true,
		},
		{
			name: "max drift set to 0 - disables drift compensation",
			config: OsqueryConfig{
				ScheduleMaxDrift: intPtr(0),
			},
			expectedDrift:  float64(0),
			expectedInJSON: true,
		},
		{
			name: "max drift does not override explicit options value",
			config: OsqueryConfig{
				Options: map[string]interface{}{
					"schedule_max_drift": 120,
				},
				ScheduleMaxDrift: intPtr(60),
			},
			expectedDrift:  float64(120),
			expectedInJSON: true,
		},
		{
			name: "both splay percent and max drift",
			config: OsqueryConfig{
				ScheduleSplayPercent: intPtr(15),
				ScheduleMaxDrift:     intPtr(90),
			},
			expectedDrift:  float64(90),
			expectedInJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := tt.config.Render()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(rendered, &result)
			require.NoError(t, err)

			if tt.expectedInJSON {
				options, ok := result["options"].(map[string]interface{})
				require.True(t, ok, "options should be present in rendered output")
				assert.Equal(t, tt.expectedDrift, options["schedule_max_drift"])
			} else {
				if options, ok := result["options"].(map[string]interface{}); ok {
					_, hasDrift := options["schedule_max_drift"]
					assert.False(t, hasDrift, "schedule_max_drift should not be in options")
				}
			}
		})
	}
}

func TestRRuleScheduleConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *RRuleScheduleConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty rrule",
			config:   &RRuleScheduleConfig{RRule: ""},
			expected: false,
		},
		{
			name:     "valid rrule",
			config:   &RRuleScheduleConfig{RRule: "FREQ=DAILY"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsEnabled())
		})
	}
}

func TestRRuleScheduleConfig_GetSplay(t *testing.T) {
	tests := []struct {
		name     string
		splay    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "empty defaults to 0s",
			splay:    "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "explicit 30s",
			splay:    "30s",
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "explicit 5m",
			splay:    "5m",
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "explicit 2h",
			splay:    "2h",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "max splay 12h",
			splay:    "12h",
			expected: 12 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "0s (disabled)",
			splay:    "0s",
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "exceeds max",
			splay:   "13h",
			wantErr: true,
		},
		{
			name:    "invalid format",
			splay:   "notaduration",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RRuleScheduleConfig{Splay: tt.splay}
			splay, err := c.GetSplay()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, splay)
			}
		})
	}
}

func TestRRuleScheduleConfig_ParseDates(t *testing.T) {
	t.Run("parse valid start date", func(t *testing.T) {
		c := &RRuleScheduleConfig{StartDate: "2024-01-01T00:00:00Z"}
		parsed, err := c.ParseStartDate()
		require.NoError(t, err)
		require.NotNil(t, parsed)
		assert.Equal(t, 2024, parsed.Year())
		assert.Equal(t, 1, int(parsed.Month()))
		assert.Equal(t, 1, parsed.Day())
	})

	t.Run("parse empty start date", func(t *testing.T) {
		c := &RRuleScheduleConfig{StartDate: ""}
		_, err := c.ParseStartDate()
		require.Error(t, err)
	})

	t.Run("parse invalid start date", func(t *testing.T) {
		c := &RRuleScheduleConfig{StartDate: "not-a-date"}
		_, err := c.ParseStartDate()
		require.Error(t, err)
	})

	t.Run("parse valid end date", func(t *testing.T) {
		c := &RRuleScheduleConfig{EndDate: "2024-12-31T23:59:59Z"}
		parsed, err := c.ParseEndDate()
		require.NoError(t, err)
		require.NotNil(t, parsed)
		assert.Equal(t, 2024, parsed.Year())
		assert.Equal(t, 12, int(parsed.Month()))
		assert.Equal(t, 31, parsed.Day())
	})

	t.Run("parse empty end date", func(t *testing.T) {
		c := &RRuleScheduleConfig{EndDate: ""}
		parsed, err := c.ParseEndDate()
		require.NoError(t, err)
		assert.Nil(t, parsed)
	})
}

func intPtr(i int) *int {
	return &i
}
