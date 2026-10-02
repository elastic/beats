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

//go:build requirefips

package tlscommon

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFIPSTLSVersion(t *testing.T) {
	// These tests are a bit verbose, but given the sensitivity to changes here, it's not a bad idea.
	tests := []struct {
		name      string
		v         uint16
		expectErr string
	}{
		{
			name:      "TLSv1.0",
			v:         tls.VersionTLS10,
			expectErr: "unsupported tls version: TLSv1.0",
		},
		{
			name:      "TLSv1.1",
			v:         tls.VersionTLS11,
			expectErr: "unsupported tls version: TLSv1.1",
		},
		{
			name: "TLSv1.2",
			v:    tls.VersionTLS12,
		},
		{
			name: "TLSv1.3",
			v:    tls.VersionTLS13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := TLSVersion(tt.v)
			if tt.expectErr != "" {
				require.EqualError(t, tv.Validate(), tt.expectErr)
			} else {
				require.NoError(t, tv.Validate())
			}
		})
	}
}
