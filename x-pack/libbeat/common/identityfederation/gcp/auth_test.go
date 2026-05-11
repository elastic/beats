// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google/externalaccount"
)

func TestParamsValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  Params
		wantErr string
	}{
		{
			name:    "missing all",
			params:  Params{},
			wantErr: "audience is required",
		},
		{
			name:    "missing audience",
			params:  Params{GlobalRoleARN: "arn", JWTFilePath: "/p"},
			wantErr: "audience is required",
		},
		{
			name:    "missing global role",
			params:  Params{Audience: "aud", JWTFilePath: "/p"},
			wantErr: "global role ARN is required",
		},
		{
			name:    "missing jwt path",
			params:  Params{Audience: "aud", GlobalRoleARN: "arn"},
			wantErr: "JWT file path is required",
		},
		{
			name: "all set",
			params: Params{
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

func TestNewTokenSourceValidatesParams(t *testing.T) {
	_, err := NewTokenSource(t.Context(), Params{})
	require.ErrorContains(t, err, "invalid GCP identity federation params")
}

func TestAwsCredentialsSupplierAwsRegion(t *testing.T) {
	s := &awsCredentialsSupplier{region: "eu-west-1"}
	got, err := s.AwsRegion(t.Context(), externalaccount.SupplierOptions{})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", got)
}
