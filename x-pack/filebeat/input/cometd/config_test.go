// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Validate that it finds the application default credentials and does
// not trigger a config validation error because credentials were not
// set in the config.
func TestConfigValidate(t *testing.T) {
	c := defaultConfig()
	assert.NoError(t, c.Validate())
}
