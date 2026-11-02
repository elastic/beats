// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		content   string
		wantPath  string
		wantField string
		wantErr   string
	}{
		{content: "app/creds#token", wantPath: "app/creds", wantField: "token"},
		{content: "/app/creds/#token", wantPath: "app/creds", wantField: "token"},
		{content: "my%20app/db%3A5432#TLS%20Cert%231", wantPath: "my app/db:5432", wantField: "TLS Cert#1"},
		{content: "kv/app#contrase%C3%B1a", wantPath: "kv/app", wantField: "contraseña"},
		{content: "app/creds", wantErr: "must have the form vault://<name>@<path>#<field>"},
		{content: "app/creds#a#b", wantErr: `field must not contain "#"`},
		{content: "#token", wantErr: "empty path or field"},
		{content: "app/creds#", wantErr: "empty path or field"},
		{content: "%2F#token", wantErr: "empty path or field"},
		{content: "app%zz#token", wantErr: "invalid percent-encoding in path"},
		{content: "app#to%zzken", wantErr: "invalid percent-encoding in field"},
		{content: "app/creds#user@example", wantErr: `must not contain "@"`},
		{content: "app/creds#user%40example", wantPath: "app/creds", wantField: "user@example"},
		{content: "%2Fapp/creds%2F#token", wantPath: "app/creds", wantField: "token"},
		{content: "a..b/c.d#token", wantPath: "a..b/c.d", wantField: "token"},
		{content: "app#..", wantPath: "app", wantField: ".."},
		{content: "../other/data/prod#password", wantErr: `path must not have "." or ".." segments`},
		{content: "app/../../other#password", wantErr: `path must not have "." or ".." segments`},
		{content: "app/./creds#password", wantErr: `path must not have "." or ".." segments`},
		{content: "app/..#password", wantErr: `path must not have "." or ".." segments`},
		{content: "..%2Fother#password", wantErr: `path must not have "." or ".." segments`},
		{content: "%2E%2E/other#password", wantErr: `path must not have "." or ".." segments`},
	}
	for _, tc := range tests {
		t.Run(tc.content, func(t *testing.T) {
			path, field, err := parseReference(tc.content)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "reference %q must be rejected", tc.content)
				return
			}
			require.NoError(t, err, "reference %q must be valid", tc.content)
			assert.Equal(t, tc.wantPath, path, "unexpected path")
			assert.Equal(t, tc.wantField, field, "unexpected field")
		})
	}
}
