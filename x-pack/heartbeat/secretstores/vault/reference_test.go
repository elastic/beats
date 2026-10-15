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
		{content: "app/creds", wantErr: "must have the form vault://<path>#<field>"},
		{content: "app/creds#a#b", wantErr: `field must not contain "#"`},
		{content: "#token", wantErr: "empty path or field"},
		{content: "app/creds#", wantErr: "empty path or field"},
		{content: "%2F#token", wantErr: "empty path or field"},
		{content: "app%zz#token", wantErr: "invalid path"},
		{content: "app#to%zzken", wantErr: "invalid field"},
		{content: "prod@app/creds#token", wantErr: "named connections are not supported yet"},
		{content: "app/creds#user%40example", wantPath: "app/creds", wantField: "user@example"},
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
