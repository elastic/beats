// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package gcppubsub

import (
	"os"
	"path/filepath"
	"testing"

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
