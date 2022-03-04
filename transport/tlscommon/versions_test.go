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

package tlscommon

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTLSVersion(t *testing.T) {
	// These tests are a bit verbose, but given the sensitivity to changes here, it's not a bad idea.
	tests := []struct {
		name string
		v    uint16
		want *TLSVersionDetails
	}{
		{
			"unknown",
			0x0,
			nil,
		},
		{
			"TLSv1.0",
			tls.VersionTLS10,
			&TLSVersionDetails{Version: "1.0", Protocol: "tls", Combined: "TLSv1.0"},
		},
		{
			"TLSv1.1",
			tls.VersionTLS11,
			&TLSVersionDetails{Version: "1.1", Protocol: "tls", Combined: "TLSv1.1"},
		},
		{
			"TLSv1.2",
			tls.VersionTLS12,
			&TLSVersionDetails{Version: "1.2", Protocol: "tls", Combined: "TLSv1.2"},
		},
		{
			"TLSv1.3",
			tls.VersionTLS13,
			&TLSVersionDetails{Version: "1.3", Protocol: "tls", Combined: "TLSv1.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := TLSVersion(tt.v)
			require.Equal(t, tt.want, tv.Details())
			if tt.want == nil {
				require.Equal(t, tt.want, tv.Details())
				require.Equal(t, tt.name, "unknown")
			} else {
				require.Equal(t, tt.name, tv.String())
			}
		})
	}
}
