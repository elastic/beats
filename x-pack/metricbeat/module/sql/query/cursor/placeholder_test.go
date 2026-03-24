// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateQueryHasCursor(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid - single cursor",
			query:   "SELECT * FROM logs WHERE id > :cursor ORDER BY id",
			wantErr: false,
		},
		{
			name:    "valid - cursor in middle",
			query:   "SELECT * FROM logs WHERE timestamp > :cursor AND status = 'active'",
			wantErr: false,
		},
		{
			name:    "valid - cursor with LIMIT",
			query:   "SELECT * FROM logs WHERE id > :cursor ORDER BY id ASC LIMIT 1000",
			wantErr: false,
		},
		{
			name:    "no cursor placeholder",
			query:   "SELECT * FROM logs WHERE id > 0",
			wantErr: true,
			errMsg:  "query must contain :cursor placeholder",
		},
		{
			name:    "multiple cursor placeholders",
			query:   "SELECT * FROM logs WHERE id > :cursor AND updated_at > :cursor",
			wantErr: true,
			errMsg:  "query must contain exactly one :cursor placeholder, found 2",
		},
		{
			name:    "similar but not cursor",
			query:   "SELECT * FROM logs WHERE id > :cursor_value",
			wantErr: true,
			errMsg:  "query must contain :cursor placeholder",
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
			errMsg:  "query must contain :cursor placeholder",
		},
		{
			name:    "ignore placeholder inside single-quoted string",
			query:   "SELECT * FROM logs WHERE note = ':cursor' AND id > :cursor",
			wantErr: false,
		},
		{
			name:    "ignore placeholder inside line comment",
			query:   "SELECT * FROM logs -- :cursor\nWHERE id > :cursor",
			wantErr: false,
		},
		{
			name:    "ignore placeholder inside block comment",
			query:   "SELECT * FROM logs /* :cursor */ WHERE id > :cursor",
			wantErr: false,
		},
		{
			name:    "only placeholder in comment is invalid",
			query:   "SELECT * FROM logs /* :cursor */ WHERE id > 0",
			wantErr: true,
			errMsg:  "query must contain :cursor placeholder",
		},
		{
			name:    "ignore placeholder inside double-quoted identifier",
			query:   `SELECT * FROM "my:cursor" WHERE id > :cursor`,
			wantErr: false,
		},
		{
			name:    "ignore placeholder inside backtick-quoted identifier",
			query:   "SELECT * FROM `my:cursor` WHERE id > :cursor",
			wantErr: false,
		},
		{
			name:    "escaped single quote does not break out of string",
			query:   "SELECT * FROM t WHERE note = 'it''s :cursor' AND id > :cursor",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQueryHasCursor(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTranslateQuery(t *testing.T) {
	baseQuery := "SELECT * FROM logs WHERE id > :cursor ORDER BY id ASC LIMIT 1000"

	tests := []struct {
		name   string
		driver string
		want   string
	}{
		{
			name:   "postgres",
			driver: "postgres",
			want:   "SELECT * FROM logs WHERE id > $1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "postgresql",
			driver: "postgresql",
			want:   "SELECT * FROM logs WHERE id > $1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "cockroachdb",
			driver: "cockroachdb",
			want:   "SELECT * FROM logs WHERE id > $1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "cockroach",
			driver: "cockroach",
			want:   "SELECT * FROM logs WHERE id > $1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "mysql",
			driver: "mysql",
			want:   "SELECT * FROM logs WHERE id > ? ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "oracle",
			driver: "oracle",
			want:   "SELECT * FROM logs WHERE id > :cursor_val ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "godror",
			driver: "godror",
			want:   "SELECT * FROM logs WHERE id > :cursor_val ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "mssql",
			driver: "mssql",
			want:   "SELECT * FROM logs WHERE id > @p1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "sqlserver",
			driver: "sqlserver",
			want:   "SELECT * FROM logs WHERE id > @p1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "unknown driver - defaults to ?",
			driver: "unknown",
			want:   "SELECT * FROM logs WHERE id > ? ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "case insensitive - POSTGRES",
			driver: "POSTGRES",
			want:   "SELECT * FROM logs WHERE id > $1 ORDER BY id ASC LIMIT 1000",
		},
		{
			name:   "case insensitive - MySQL",
			driver: "MySQL",
			want:   "SELECT * FROM logs WHERE id > ? ORDER BY id ASC LIMIT 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateQuery(baseQuery, tt.driver)
			assert.Equal(t, tt.want, result)
		})
	}

	t.Run("does not replace placeholder inside string or comments", func(t *testing.T) {
		query := "SELECT ':cursor' AS s /* :cursor */ FROM logs -- :cursor\nWHERE id > :cursor"
		got := TranslateQuery(query, "postgres")
		want := "SELECT ':cursor' AS s /* :cursor */ FROM logs -- :cursor\nWHERE id > $1"
		assert.Equal(t, want, got)
	})
}

func TestCountPlaceholders(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{
			name:  "no cursor",
			query: "SELECT * FROM logs",
			want:  0,
		},
		{
			name:  "one cursor",
			query: "SELECT * FROM logs WHERE id > :cursor",
			want:  1,
		},
		{
			name:  "two cursors",
			query: "SELECT * FROM logs WHERE id > :cursor AND ts > :cursor",
			want:  2,
		},
		{
			name:  "cursor_value not matched",
			query: "SELECT * FROM logs WHERE id > :cursor_value",
			want:  0,
		},
		{
			name:  "ignore placeholders in string and comments",
			query: "SELECT ':cursor' /* :cursor */ FROM logs -- :cursor\nWHERE id > :cursor",
			want:  1,
		},
		{
			name:  "ignore placeholder in double-quoted identifier",
			query: `SELECT * FROM "my:cursor" WHERE id > :cursor`,
			want:  1,
		},
		{
			name:  "ignore placeholder in backtick-quoted identifier",
			query: "SELECT * FROM `my:cursor` WHERE id > :cursor",
			want:  1,
		},
		{
			name:  "escaped single quote does not break string",
			query: "SELECT * FROM t WHERE note = 'it''s :cursor' AND id > :cursor",
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CountPlaceholders(tt.query))
		})
	}
}
