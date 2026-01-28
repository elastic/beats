// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	confpkg "github.com/elastic/elastic-agent-libs/config"
)

func Test_validateConfig(t *testing.T) {
	testCases := []struct {
		name      string // Sub-test name.
		config    config // Load config parameters.
		wantError string // Expected error
	}{
		{
			name: "empty URL",
			config: config{
				URL:          "",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
			},
			wantError: "string value is not set accessing 'url'",
		},
		{
			name: "invalid method",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       "random",
			},
			wantError: "method must be POST, PUT or PATCH: random",
		},
		{
			name: "invalid ResponseBody",
			config: config{
				URL:          "/",
				ResponseBody: "",
				Method:       http.MethodPost,
			},
			wantError: "response_body must be valid JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := confpkg.MustNewConfigFrom(tc.config)
			config := defaultConfig()
			err := c.Unpack(&config)

			// Validate responses
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tc.wantError)
			}
		})
	}
}

func TestApplyInFlightDefaults(t *testing.T) {
	tests := []struct {
		name              string
		maxInFlight       int64
		highWaterInFlight int64
		lowWaterInFlight  int64
		wantHighWater     int64
		wantLowWater      int64
	}{
		{
			name:          "max only - high water defaults to 50%",
			maxInFlight:   1000000,
			wantHighWater: 500000,
			wantLowWater:  400000, // 80% of 500000
		},
		{
			name:          "max only - small value uses 64KB offset",
			maxInFlight:   200000,
			wantHighWater: 100000,
			wantLowWater:  100000 - 64*1024, // high_water - 64KB < 80%
		},
		{
			name:              "all values explicit",
			maxInFlight:       1000000,
			highWaterInFlight: 700000,
			lowWaterInFlight:  600000,
			wantHighWater:     700000,
			wantLowWater:      600000,
		},
		{
			name:              "high water explicit, low water defaults",
			maxInFlight:       1000000,
			highWaterInFlight: 800000,
			wantHighWater:     800000,
			wantLowWater:      640000, // 80% of 800000
		},
		{
			name:          "no max - no defaults applied",
			maxInFlight:   0,
			wantHighWater: 0,
			wantLowWater:  0,
		},
		{
			name:          "very small max - low water clamped to 0",
			maxInFlight:   1000,
			wantHighWater: 500,
			wantLowWater:  0, // 500 - 64KB would be negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				MaxInFlight:       tt.maxInFlight,
				HighWaterInFlight: tt.highWaterInFlight,
				LowWaterInFlight:  tt.lowWaterInFlight,
			}
			c.applyInFlightDefaults()
			assert.Equal(t, tt.wantHighWater, c.HighWaterInFlight)
			assert.Equal(t, tt.wantLowWater, c.LowWaterInFlight)
		})
	}
}

func TestValidateInFlightLimits(t *testing.T) {
	tests := []struct {
		name              string
		maxInFlight       int64
		highWaterInFlight int64
		lowWaterInFlight  int64
		wantError         string
	}{
		{
			name:              "valid configuration",
			maxInFlight:       1000,
			highWaterInFlight: 800,
			lowWaterInFlight:  500,
			wantError:         "",
		},
		{
			name:              "no limits - valid",
			maxInFlight:       0,
			highWaterInFlight: 0,
			lowWaterInFlight:  0,
			wantError:         "",
		},
		{
			name:        "negative max",
			maxInFlight: -100,
			wantError:   "max_in_flight_bytes is negative",
		},
		{
			name:              "negative high water",
			maxInFlight:       1000,
			highWaterInFlight: -100,
			wantError:         "high_water_in_flight_bytes is negative",
		},
		{
			name:              "negative low water",
			maxInFlight:       1000,
			highWaterInFlight: 800,
			lowWaterInFlight:  -100,
			wantError:         "low_water_in_flight_bytes is negative",
		},
		{
			name:              "high water >= max",
			maxInFlight:       1000,
			highWaterInFlight: 1000,
			lowWaterInFlight:  500,
			wantError:         "high_water_in_flight_bytes (1000) must be less than max_in_flight_bytes (1000)",
		},
		{
			name:              "low water >= high water",
			maxInFlight:       1000,
			highWaterInFlight: 800,
			lowWaterInFlight:  800,
			wantError:         "low_water_in_flight_bytes (800) must be less than high_water_in_flight_bytes (800)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				MaxInFlight:       tt.maxInFlight,
				HighWaterInFlight: tt.highWaterInFlight,
				LowWaterInFlight:  tt.lowWaterInFlight,
			}
			err := c.validateInFlightLimits()
			if tt.wantError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}
