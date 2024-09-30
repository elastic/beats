// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

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
		"No filebeat section": {
			c:           &Config{Beatconfig: map[string]interface{}{"other": map[string]interface{}{}}},
			hasError:    true,
			errorString: "Configuration key 'filebeat' is required",
		},
		"Valid config": {
			c:           &Config{Beatconfig: map[string]interface{}{"filebeat": map[string]interface{}{}}},
			hasError:    false,
			errorString: "",
		},
	}
	for name, tc := range tests {
		err := tc.c.Validate()
		if tc.hasError {
			assert.NotNil(t, err, name)
			assert.Equal(t, err.Error(), tc.errorString, name)
		}
		if !tc.hasError {
			assert.Nil(t, err, name)
		}
	}
}
