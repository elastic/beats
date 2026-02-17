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

func TestOsqueryConfig_Render_Options(t *testing.T) {
	// Native osquery options (e.g. schedule_splay_percent, schedule_max_drift) are
	// passed through in Options; defaults are applied when building config for osqueryd (beater).
	rendered, err := OsqueryConfig{
		Options: map[string]interface{}{
			"schedule_splay_percent": 10,
			"schedule_max_drift":     60,
		},
	}.Render()
	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(rendered, &result)
	require.NoError(t, err)
	options, ok := result["options"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), options["schedule_splay_percent"])
	assert.Equal(t, float64(60), options["schedule_max_drift"])
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
