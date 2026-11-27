// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

func TestNewCredential(t *testing.T) {
	cfg := Config{
		TenantId:     "00000000-0000-0000-0000-000000000000",
		ClientId:     "00000000-0000-0000-0000-000000000001",
		ClientSecret: "secret",
	}

	t.Run("client secret", func(t *testing.T) {
		cred, err := NewCredential(cfg, policy.ClientOptions{}, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err, "expected the client secret credential to be created")
		assert.IsType(t, &azidentity.ClientSecretCredential{}, cred, "expected a client secret credential without use_cloud_connectors")
	})

	t.Run("cloud connectors with WII environment", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "https://wii.example.com")
		t.Setenv(identityfederation.WIISSLCertFileEnvVar, "/cert")
		t.Setenv(identityfederation.WIISSLKeyFileEnvVar, "/key")
		cfg := cfg
		cfg.UseCloudConnectors = true

		cred, err := NewCredential(cfg, policy.ClientOptions{}, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err, "expected the WII credential to be created")
		assert.IsType(t, &azidentity.ClientAssertionCredential{}, cred, "expected a client assertion credential with use_cloud_connectors")
	})

	t.Run("cloud connectors without WII environment", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		cfg := cfg
		cfg.UseCloudConnectors = true

		_, err := NewCredential(cfg, policy.ClientOptions{}, logptest.NewTestingLogger(t, ""))
		assert.ErrorContains(t, err, identityfederation.WIIIssuerURLEnvVar, "expected an error that names the missing WII variable, not a client secret fallback")
	})
}
