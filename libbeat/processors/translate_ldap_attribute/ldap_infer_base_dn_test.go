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

//go:build !requirefips

package translate_ldap_attribute

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferBaseDNFromAddress(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		expectDN  string
		expectErr bool
	}{
		{
			name:      "Skip first label with 3 parts",
			address:   "ldap://dc1.example.com:389",
			expectDN:  "dc=example,dc=com",
			expectErr: false,
		},
		{
			name:      "Two part domain no skip",
			address:   "ldaps://example.com:636",
			expectDN:  "dc=example,dc=com",
			expectErr: false,
		},
		{
			name:      "Multi part domain co.uk",
			address:   "ldap://auth.example.co.uk:389",
			expectDN:  "dc=example,dc=co,dc=uk",
			expectErr: false,
		},
		{
			name:      "Hostname only (no domain)",
			address:   "ldap://localhost:389",
			expectErr: true,
		},
		{
			name:      "IPv4 address (cannot infer)",
			address:   "ldaps://192.168.1.10:636",
			expectErr: true,
		},
		{
			name:      "IPv6 address (cannot infer)",
			address:   "ldap://[2001:db8::1]:389",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &ldapClient{ldapConfig: &ldapConfig{address: tt.address}, log: logp.NewLogger("test")}
			err := client.inferBaseDNFromAddress()
			if tt.expectErr {
				require.Error(t, err)
				assert.Empty(t, client.baseDN)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectDN, client.baseDN)
			}
		})
	}
}
