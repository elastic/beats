// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	confpkg "github.com/elastic/elastic-agent-libs/config"
)

func Test_validateConfig(t *testing.T) {
	testCases := []struct {
		name      string // Sub-test name.
		config    config // Load config parameters.
		wantError string // Expected error
	}{
		{
			name: "empty URL",
			config: config{
				URL:          "",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
			},
			wantError: "string value is not set accessing 'url'",
		},
		{
			name: "invalid method",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       "random",
			},
			wantError: "method must be POST, PUT or PATCH: random",
		},
		{
			name: "invalid ResponseBody",
			config: config{
				URL:          "/",
				ResponseBody: "",
				Method:       http.MethodPost,
			},
			wantError: "response_body must be valid JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := confpkg.MustNewConfigFrom(tc.config)
			config := defaultConfig()
			err := c.Unpack(&config)

			// Validate responses
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tc.wantError)
			}
		})
	}
}
