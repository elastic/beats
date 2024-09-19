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
		"No Items": {
			c:           &Config{Beatconfig: map[string]interface{}{}},
			hasError:    true,
			errorString: "NO config sent",
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
