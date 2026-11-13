// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var tests = []struct {
	description   string
	host          string
	url           string
	convertedHost string
	result        bool
}{
	{
		"Validate if neither urls nor host specified returns error",
		"",
		"",
		"",
		false,
	},
	{
		"Validate if only urls are present then the config is moved to hosts",
		"",
		"http://localhost:8080",
		"http://localhost:8080",
		true,
	},
	{
		"Validate if only hosts are present then the config is valid",
		"http://localhost:8080",
		"",
		"http://localhost:8080",
		true,
	},
	{
		"Validate if no scheme is present then it is added correctly",
		"localhost",
		"",
		"http://localhost",
		true,
	},
	{
		"Validate if no scheme is present but has a port then it is added correctly",
		"localhost:8080",
		"",
		"http://localhost:8080",
		true,
	},
	{
		"Validate if schemes like unix are honored",
		"unix://localhost:8080",
		"",
		"unix://localhost:8080",
		true,
	},
}

func TestResponseConfigValidateBodyMaxBytes(t *testing.T) {
	base := responseConfig{
		IncludeBody:         "never",
		IncludeBodyMaxBytes: 2048,
	}

	for _, tc := range []struct {
		name      string
		value     int
		wantError bool
	}{
		{"positive value is valid", 1, false},
		{"default 4MiB is valid", 4 * 1024 * 1024, false},
		{"zero is invalid", 0, true},
		{"negative is invalid", -1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.CheckBodyMaxBytes = tc.value
			err := cfg.Validate()
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			config := Config{}
			if test.host != "" {
				config.Hosts = []string{test.host}
			}

			if test.url != "" {
				config.URLs = []string{test.url}
			}

			err := config.Validate()
			if test.result {
				assert.NoError(t, err)
				assert.Equal(t, test.convertedHost, config.Hosts[0])
			} else {
				assert.Error(t, err)
			}

		})
	}
}
