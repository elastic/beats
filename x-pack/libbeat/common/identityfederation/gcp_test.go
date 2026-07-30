// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google/externalaccount"
)

func TestGCPParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  GCPParams
		wantErr string
	}{
		{
			name:    "missing all",
			params:  GCPParams{},
			wantErr: "audience is required",
		},
		{
			name:    "missing audience",
			params:  GCPParams{GlobalRoleARN: "arn", JWTFilePath: "/p"},
			wantErr: "audience is required",
		},
		{
			name:    "missing global role",
			params:  GCPParams{Audience: "aud", JWTFilePath: "/p"},
			wantErr: "global role ARN is required",
		},
		{
			name:    "missing jwt path",
			params:  GCPParams{Audience: "aud", GlobalRoleARN: "arn"},
			wantErr: "JWT file path is required",
		},
		{
			name: "all set",
			params: GCPParams{
				Audience:      "aud",
				GlobalRoleARN: "arn",
				JWTFilePath:   "/p",
			},
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

func TestGCPNewTokenSourceValidatesParams(t *testing.T) {
	_, err := GCPNewTokenSource(t.Context(), GCPParams{})
	require.ErrorContains(t, err, "invalid GCP identity federation params")
}

func TestAWSCredentialsSupplierAwsRegion(t *testing.T) {
	s := &awsCredentialsSupplier{region: "eu-west-1"}
	got, err := s.AwsRegion(t.Context(), externalaccount.SupplierOptions{})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", got)
}
