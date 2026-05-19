// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJWT is a syntactically valid (3 dot-separated parts) JWT for tests.
const fakeJWT = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature"

func TestAzureParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  AzureParams
		wantErr string
	}{
		{
			name:    "missing all",
			params:  AzureParams{},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing tenant id",
			params:  AzureParams{ClientID: "client", JWTFilePath: "/p"},
			wantErr: "TenantID is required",
		},
		{
			name:    "missing client id",
			params:  AzureParams{TenantID: "tenant", JWTFilePath: "/p"},
			wantErr: "ClientID is required",
		},
		{
			name:    "missing jwt path",
			params:  AzureParams{TenantID: "tenant", ClientID: "client"},
			wantErr: "JWTFilePath is required",
		},
		{
			name:   "all set",
			params: AzureParams{TenantID: "tenant", ClientID: "client", JWTFilePath: "/p"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAzureReadJWT(t *testing.T) {
	dir := t.TempDir()

	t.Run("happy path", func(t *testing.T) {
		p := filepath.Join(dir, "good")
		require.NoError(t, os.WriteFile(p, []byte("  "+fakeJWT+"\n"), 0o600))
		got, err := AzureReadJWT(p)
		require.NoError(t, err)
		assert.Equal(t, fakeJWT, got)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := AzureReadJWT(filepath.Join(dir, "does-not-exist"))
		require.ErrorContains(t, err, "reading JWT file")
	})

	t.Run("empty file", func(t *testing.T) {
		p := filepath.Join(dir, "empty")
		require.NoError(t, os.WriteFile(p, []byte("   \n"), 0o600))
		_, err := AzureReadJWT(p)
		require.ErrorContains(t, err, "is empty")
	})

	t.Run("malformed jwt", func(t *testing.T) {
		p := filepath.Join(dir, "malformed")
		require.NoError(t, os.WriteFile(p, []byte("not.a-valid-jwt"), 0o600))
		_, err := AzureReadJWT(p)
		require.ErrorContains(t, err, "invalid JWT")
	})
}

func TestAzureNewClientAssertionCredential(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "jwt")
		require.NoError(t, os.WriteFile(p, []byte(fakeJWT), 0o600))

		cred, err := AzureNewClientAssertionCredential(AzureParams{
			TenantID:    "00000000-0000-0000-0000-000000000000",
			ClientID:    "00000000-0000-0000-0000-000000000001",
			JWTFilePath: p,
		})
		require.NoError(t, err)
		require.NotNil(t, cred)
	})

	t.Run("invalid params", func(t *testing.T) {
		_, err := AzureNewClientAssertionCredential(AzureParams{})
		require.ErrorContains(t, err, "invalid Azure identity federation params")
	})
}
