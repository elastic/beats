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

package oteltranslate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestTLSCommonToOTel(t *testing.T) {

	t.Run("when ssl.enabled = false", func(t *testing.T) {
		b := false
		input := &tlscommon.Config{
			Enabled: &b,
		}
		got, err := TLSCommonToOTel(input)
		require.NoError(t, err)
		want := map[string]any{
			"insecure": true,
		}

		assert.Equal(t, want, got)
	})

	tests := []struct {
		name  string
		input *tlscommon.Config
		want  map[string]any
		err   bool
	}{
		{
			name: "when unsupported configuration is passed",
			input: &tlscommon.Config{
				CATrustedFingerprint: "a3:5f:bf:93:12:8f:bc:5c:ab:14:6d:bf:e4:2a:7f:98:9d:2f:16:92:76:c4:12:ab:67:89:fc:56:4b:8e:0c:43",
			},
			want: nil,
			err:  true,
		},
		{
			name: "when ssl.verification_mode:none ",
			input: &tlscommon.Config{
				VerificationMode: tlscommon.VerifyNone,
			},
			want: map[string]any{
				"insecure_skip_verify":         true,
				"include_system_ca_certs_pool": true,
			},
			err: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := TLSCommonToOTel(test.input)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, got, "beats to otel ssl mapping")
			}

		})
	}
}
