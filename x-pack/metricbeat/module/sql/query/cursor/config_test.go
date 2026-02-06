// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled cursor - no validation",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid integer cursor",
			config: Config{
				Enabled: true,
				Column:  "id",
				Type:    CursorTypeInteger,
				Default: "0",
			},
			wantErr: false,
		},
		{
			name: "valid timestamp cursor",
			config: Config{
				Enabled: true,
				Column:  "created_at",
				Type:    CursorTypeTimestamp,
				Default: "2024-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "valid date cursor",
			config: Config{
				Enabled: true,
				Column:  "event_date",
				Type:    CursorTypeDate,
				Default: "2024-01-01",
			},
			wantErr: false,
		},
		{
			name: "missing column",
			config: Config{
				Enabled: true,
				Column:  "",
				Type:    CursorTypeInteger,
				Default: "0",
			},
			wantErr: true,
			errMsg:  "cursor.column is required",
		},
		{
			name: "invalid type",
			config: Config{
				Enabled: true,
				Column:  "id",
				Type:    "invalid",
				Default: "0",
			},
			wantErr: true,
			errMsg:  "cursor.type must be",
		},
		{
			name: "empty type",
			config: Config{
				Enabled: true,
				Column:  "id",
				Type:    "",
				Default: "0",
			},
			wantErr: true,
			errMsg:  "cursor.type must be",
		},
		{
			name: "missing default",
			config: Config{
				Enabled: true,
				Column:  "id",
				Type:    CursorTypeInteger,
				Default: "",
			},
			wantErr: true,
			errMsg:  "cursor.default is required",
		},
		{
			name: "invalid integer default",
			config: Config{
				Enabled: true,
				Column:  "id",
				Type:    CursorTypeInteger,
				Default: "not-an-integer",
			},
			wantErr: true,
			errMsg:  "cursor.default is invalid",
		},
		{
			name: "invalid timestamp default",
			config: Config{
				Enabled: true,
				Column:  "created_at",
				Type:    CursorTypeTimestamp,
				Default: "not-a-timestamp",
			},
			wantErr: true,
			errMsg:  "cursor.default is invalid",
		},
		{
			name: "invalid date default",
			config: Config{
				Enabled: true,
				Column:  "event_date",
				Type:    CursorTypeDate,
				Default: "not-a-date",
			},
			wantErr: true,
			errMsg:  "cursor.default is invalid",
		},
		// Float cursor tests
		{
			name: "valid float cursor",
			config: Config{
				Enabled: true,
				Column:  "score",
				Type:    CursorTypeFloat,
				Default: "0.0",
			},
			wantErr: false,
		},
		{
			name: "invalid float default",
			config: Config{
				Enabled: true,
				Column:  "score",
				Type:    CursorTypeFloat,
				Default: "not-a-float",
			},
			wantErr: true,
			errMsg:  "cursor.default is invalid",
		},
		// Decimal cursor tests
		{
			name: "valid decimal cursor",
			config: Config{
				Enabled: true,
				Column:  "price",
				Type:    CursorTypeDecimal,
				Default: "99.95",
			},
			wantErr: false,
		},
		{
			name: "invalid decimal default",
			config: Config{
				Enabled: true,
				Column:  "price",
				Type:    CursorTypeDecimal,
				Default: "not-a-decimal",
			},
			wantErr: true,
			errMsg:  "cursor.default is invalid",
		},
		// Direction tests
		{
			name: "valid direction asc",
			config: Config{
				Enabled:   true,
				Column:    "id",
				Type:      CursorTypeInteger,
				Default:   "0",
				Direction: CursorDirectionAsc,
			},
			wantErr: false,
		},
		{
			name: "valid direction desc",
			config: Config{
				Enabled:   true,
				Column:    "id",
				Type:      CursorTypeInteger,
				Default:   "99999",
				Direction: CursorDirectionDesc,
			},
			wantErr: false,
		},
		{
			name: "empty direction defaults to asc",
			config: Config{
				Enabled:   true,
				Column:    "id",
				Type:      CursorTypeInteger,
				Default:   "0",
				Direction: "",
			},
			wantErr: false,
		},
		{
			name: "invalid direction",
			config: Config{
				Enabled:   true,
				Column:    "id",
				Type:      CursorTypeInteger,
				Default:   "0",
				Direction: "sideways",
			},
			wantErr: true,
			errMsg:  "cursor.direction must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigValidate_DirectionWithAllTypes(t *testing.T) {
	// Verify direction works with every cursor type
	types := []struct {
		cursorType   string
		defaultValue string
	}{
		{CursorTypeInteger, "0"},
		{CursorTypeTimestamp, "2024-01-01T00:00:00Z"},
		{CursorTypeDate, "2024-01-01"},
		{CursorTypeFloat, "0.0"},
		{CursorTypeDecimal, "0.00"},
	}

	for _, tt := range types {
		for _, dir := range []string{CursorDirectionAsc, CursorDirectionDesc} {
			name := tt.cursorType + "_" + dir
			t.Run(name, func(t *testing.T) {
				cfg := Config{
					Enabled:   true,
					Column:    "col",
					Type:      tt.cursorType,
					Default:   tt.defaultValue,
					Direction: dir,
				}
				err := cfg.Validate()
				require.NoError(t, err, "direction=%s with type=%s should be valid", dir, tt.cursorType)
			})
		}
	}
}

func TestIsValidCursorType(t *testing.T) {
	tests := []struct {
		cursorType string
		valid      bool
	}{
		{CursorTypeInteger, true},
		{CursorTypeTimestamp, true},
		{CursorTypeDate, true},
		{CursorTypeFloat, true},
		{CursorTypeDecimal, true},
		{"", false},
		{"string", false},
		{"INTEGER", false}, // case sensitive
		{"FLOAT", false},   // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.cursorType, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidCursorType(tt.cursorType))
		})
	}
}
