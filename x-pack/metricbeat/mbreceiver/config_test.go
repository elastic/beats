// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		c           *Config
		hasError    bool
		errorString string
	}{
		"Empty config": {
			c:           &Config{Beatconfig: map[string]interface{}{}},
			hasError:    true,
			errorString: "Configuration is required",
		},
		"No metricbeat section": {
			c:           &Config{Beatconfig: map[string]interface{}{"other": map[string]interface{}{}}},
			hasError:    true,
			errorString: "Configuration key 'metricbeat' is required",
		},
		"Valid config": {
			c:           &Config{Beatconfig: map[string]interface{}{"metricbeat": map[string]interface{}{}}},
			hasError:    false,
			errorString: "",
		},
	}
	for name, tc := range tests {
		err := tc.c.Validate()
		if tc.hasError {
			assert.NotNilf(t, err, "%s failed, should have had error", name)
			assert.Equalf(t, err.Error(), tc.errorString, "%s failed, error not equal", name)
		} else {
			assert.Nilf(t, err, "%s failed, should not have error", name)
		}
	}
}
