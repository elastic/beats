// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		content   string
		wantRef   accountRef
		wantField string
		wantErr   string
	}{
		{content: "MySafe/MyObject#Content", wantRef: accountRef{safe: "MySafe", object: "MyObject"}, wantField: "Content"},
		{content: "/MySafe/MyObject/#password", wantRef: accountRef{safe: "MySafe", object: "MyObject"}, wantField: "password"},
		{content: "MySafe/Root/MyObject#UserName", wantRef: accountRef{safe: "MySafe", folder: "Root", object: "MyObject"}, wantField: "UserName"},
		{content: "My%20Safe/db%3A5432#TLS%20Cert%231", wantRef: accountRef{safe: "My Safe", object: "db:5432"}, wantField: "TLS Cert#1"},
		{content: "MySafe/MyObject#user%40example", wantRef: accountRef{safe: "MySafe", object: "MyObject"}, wantField: "user@example"},
		{content: "prod@MySafe/MyObject#password", wantErr: "named connections are not supported yet"},
		{content: "MySafe/MyObject", wantErr: "must have the form cyberark://<safe>/<object>#<field>"},
		{content: "OnlySafe#password", wantErr: "must have the form cyberark://<safe>/<object>#<field>"},
		{content: "a/b/c/d#password", wantErr: "must have the form cyberark://<safe>/<object>#<field>"},
		{content: "MySafe/MyObject#a#b", wantErr: `field must not contain "#"`},
		{content: "#password", wantErr: "must have the form cyberark://<safe>/<object>#<field>"},
		{content: "MySafe/MyObject#", wantErr: "empty path or field"},
		{content: "MySafe/%2F#password", wantErr: "empty path or field"},
		{content: "MySafe%zz/MyObject#password", wantErr: "invalid path"},
		{content: "MySafe/MyObject#pa%zz", wantErr: "invalid field"},
		{content: "../MyObject#password", wantErr: `path must not have "." or ".." segments`},
		{content: "MySafe/../MyObject#password", wantErr: `path must not have "." or ".." segments`},
		{content: "MySafe/./MyObject#password", wantErr: `path must not have "." or ".." segments`},
		{content: "MySafe/%2e%2e/MyObject#password", wantErr: `path must not have "." or ".." segments`},
		{content: "MySafe/sub%2Fdir#password", wantErr: `path must not have "." or ".." segments`},
	}
	for _, tc := range tests {
		t.Run(tc.content, func(t *testing.T) {
			ref, field, err := parseReference(tc.content)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "reference %q must be rejected", tc.content)
				return
			}
			require.NoError(t, err, "reference %q must be valid", tc.content)
			assert.Equal(t, tc.wantRef, ref, "unexpected account reference")
			assert.Equal(t, tc.wantField, field, "unexpected field")
		})
	}
}
