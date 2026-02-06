// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"

	confpkg "github.com/elastic/elastic-agent-libs/config"
)

func Test_validateConfig(t *testing.T) {
	testCases := []struct {
		name      string // Sub-test name.
		config    config // Load config parameters.
		wantError error  // Expected error
	}{
		{
			name: "empty URL",
			config: config{
				URL:          "",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
			},
			wantError: errors.New("string value is not set accessing 'url'"),
		},
		{
			name: "invalid method",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       "random",
			},
			wantError: errors.New("method must be POST, PUT or PATCH: random accessing config"),
		},
		{
			name: "invalid ResponseBody",
			config: config{
				URL:          "/",
				ResponseBody: "",
				Method:       http.MethodPost,
			},
			wantError: errors.New("response_body must be valid JSON accessing config"),
		},
		{
			name: "valid log destination",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
				Tracer:       &tracerConfig{Enabled: ptrTo(true), Logger: lumberjack.Logger{Filename: "http_endpoint/log"}},
			},
		},
		{
			name: "invalid log destination",
			config: config{
				URL:          "/",
				ResponseBody: `{"message": "success"}`,
				Method:       http.MethodPost,
				Tracer:       &tracerConfig{Enabled: ptrTo(true), Logger: lumberjack.Logger{Filename: "/var/log"}},
			},
			wantError: fmt.Errorf(`request tracer path must be within %q path accessing config`, inputName),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := confpkg.MustNewConfigFrom(tc.config)
			config := defaultConfig()
			err := c.Unpack(&config)

			if !sameError(err, tc.wantError) {
				t.Errorf("unexpected error from validation: got:%s want:%s", err, tc.wantError)
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}

func ptrTo[T any](v T) *T { return &v }
