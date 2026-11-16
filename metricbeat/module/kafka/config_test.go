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

package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsetConfigValidateCredentials(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		mechanism string
		wantErr   bool
		errSubstr string
	}{
		{name: "username and password", username: "user", password: "secret"},
		{name: "empty credentials"},
		{name: "OAUTHBEARER no credentials", mechanism: "OAUTHBEARER"},
		{name: "username without password", username: "user", wantErr: true, errSubstr: "password must be set"},
		{name: "password without username", password: "secret", wantErr: true, errSubstr: "username must be set"},
		{name: "OAUTHBEARER with username", mechanism: "OAUTHBEARER", username: "user", password: "secret", wantErr: true, errSubstr: "OAUTHBEARER"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig
			cfg.Username = tc.username
			cfg.Password = tc.password
			cfg.Sasl.SaslMechanism = tc.mechanism

			err := cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.ErrorContains(t, err, tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
