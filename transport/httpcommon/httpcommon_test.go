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

package httpcommon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestUnpack(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected HTTPTransportSettings
	}{
		"blank": {
			input:    "",
			expected: HTTPTransportSettings{},
		},
		"idleConnectionTimeout": {
			input: `
idle_connection_timeout: 15s
`,
			expected: HTTPTransportSettings{IdleConnTimeout: 15 * time.Second},
		},
		"timeoutAndIdleConnectionTimeout": {
			input: `
idle_connection_timeout: 15s
timeout: 5s
`,
			expected: HTTPTransportSettings{
				IdleConnTimeout: 15 * time.Second,
				Timeout:         5 * time.Second,
			},
		},
		"ssl": {
			input: `
ssl:
  verification_mode: certificate
`,
			expected: HTTPTransportSettings{
				TLS: &tlscommon.Config{
					VerificationMode: tlscommon.VerifyCertificate,
				},
			},
		},
		"complex": {
			input: `
timeout: 5s
idle_connection_timeout: 15s
ssl:
  verification_mode: certificate
`,
			expected: HTTPTransportSettings{
				TLS: &tlscommon.Config{
					VerificationMode: tlscommon.VerifyCertificate,
				},
				IdleConnTimeout: 15 * time.Second,
				Timeout:         5 * time.Second,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(tc.input)
			require.NoError(t, err)

			settings := HTTPTransportSettings{}
			err = cfg.Unpack(&settings)
			require.NoError(t, err)

			require.Equal(t, tc.expected, settings)
		})
	}
}
