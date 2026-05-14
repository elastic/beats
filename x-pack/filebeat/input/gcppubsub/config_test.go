// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package gcppubsub

import (
	"os"
	"path/filepath"
	"testing"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"
)

//nolint:gosec // false positive
const googleApplicationCredentialsVar = "GOOGLE_APPLICATION_CREDENTIALS"

func TestConfigValidateGoogleAppDefaultCreds(t *testing.T) {
	// Return the environment variables to their original state.
	original, found := os.LookupEnv(googleApplicationCredentialsVar)
	defer func() {
		if found {
			os.Setenv(googleApplicationCredentialsVar, original)
		} else {
			os.Unsetenv(googleApplicationCredentialsVar)
		}
	}()

	// Validate that it finds the application default credentials and does
	// not trigger a config validation error because credentials were not
	// set in the config.
	os.Setenv(googleApplicationCredentialsVar, filepath.Clean("testdata/fake.json"))
	c := defaultConfig()
	assert.NoError(t, c.Validate())
}

func TestConfigAPIEndpoint(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]interface{}{
		"project_id":       "test-project",
		"topic":            "test-topic",
		"subscription":     map[string]interface{}{"name": "test-sub"},
		"api_endpoint":     "custom-endpoint.googleapis.com:443",
		"credentials_file": filepath.Clean("testdata/fake.json"),
	})
	assert.NoError(t, err, "failed to create config from map")

	c := defaultConfig()
	err = cfg.Unpack(&c)
	assert.NoError(t, err, "failed to unpack config")
	assert.Equal(t, "custom-endpoint.googleapis.com:443", c.APIEndpoint, "APIEndpoint does not match expected value")
}
