// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

// TestNewAcceptsUnsupportedSettingsWithoutFIPS checks that builds without the
// requirefips tag accept the settings that FIPS builds reject.
func TestNewAcceptsUnsupportedSettingsWithoutFIPS(t *testing.T) {
	for _, tc := range unsupportedCases {
		t.Run(tc.name, func(t *testing.T) {
			err := newWithUnsupportedConfig(t, tc.osquery)

			assert.NoError(t, err, "builds without requirefips must accept %s at startup", tc.setting)
		})
	}
}

// TestValidatePolicyAcceptsUnsupportedSettingsWithoutFIPS checks that builds
// without the requirefips tag accept the settings that FIPS builds reject.
func TestValidatePolicyAcceptsUnsupportedSettingsWithoutFIPS(t *testing.T) {
	for _, tc := range unsupportedCases {
		t.Run(tc.name, func(t *testing.T) {
			inputs := unsupportedInputs(t, tc.osquery)

			err := validatePolicy(t.Context(), logptest.NewTestingLogger(t, ""), inputs)

			assert.NoError(t, err, "builds without requirefips must accept %s in a policy", tc.setting)
		})
	}
}
