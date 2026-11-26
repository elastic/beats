// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestNewRejectsUnsupportedSettings(t *testing.T) {
	for _, tc := range unsupportedCases {
		t.Run(tc.name, func(t *testing.T) {
			err := newWithUnsupportedConfig(t, tc.osquery)

			assert.ErrorContains(t, err, tc.setting, "the startup configuration must be rejected, naming the setting")
			assert.ErrorContains(t, err, "not supported in FIPS builds", "the error must say why the configuration is rejected")
		})
	}
}

func TestValidatePolicyRejectsUnsupportedSettings(t *testing.T) {
	for _, tc := range unsupportedCases {
		t.Run(tc.name, func(t *testing.T) {
			inputs := unsupportedInputs(t, tc.osquery)

			err := validatePolicy(t.Context(), logptest.NewTestingLogger(t, ""), inputs)

			assert.ErrorContains(t, err, tc.setting, "a policy update must be rejected, naming the setting")
			assert.ErrorContains(t, err, "not supported in FIPS builds", "the error must say why the policy is rejected")
		})
	}
}
