// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateConfig(t *testing.T) {
	testCases := []struct {
		name      string // Sub-test name.
		config    config // Load config parameters.
		wantError error  // Expected error
	}{
		{
			name: "invalid URL",
			config: config{
				URL:          "",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
			},
			wantError: fmt.Errorf("webhook path URL can not be empty"),
		},
		{
			name: "invalid method",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       "random",
			},
			wantError: fmt.Errorf("method must be POST, PUT or PATCH: random"),
		},
		{
			name: "invalid URL",
			config: config{
				URL:          "/",
				ResponseBody: "",
				Method:       http.MethodPost,
			},
			wantError: fmt.Errorf("response_body must be valid JSON"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.URL = ""
			// Execute config validation
			err := tc.config.Validate()

			// Validate responses
			assert.Equal(t, tc.wantError, err)
		})
	}
}
