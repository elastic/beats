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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestFindLogonServerPreservesHostname(t *testing.T) {
	t.Setenv("LOGONSERVER", "\\\\DC01")

	originalResolver := resolveTCPAddr
	resolveTCPAddr = func(network, address string) (*net.TCPAddr, error) {
		return &net.TCPAddr{IP: net.ParseIP("192.0.2.10")}, nil
	}
	t.Cleanup(func() { resolveTCPAddr = originalResolver })

	log := logp.NewLogger("test")
	addresses := findLogonServer(true, log)
	require.Len(t, addresses, 2)
	assert.Equal(t, "ldaps://DC01:636", addresses[0])
	assert.Equal(t, "ldaps://192.0.2.10:636", addresses[1])
}

func TestFindLogonServerFallsBackWithoutResolution(t *testing.T) {
	t.Setenv("LOGONSERVER", "\\\\DC02")

	originalResolver := resolveTCPAddr
	resolveTCPAddr = func(network, address string) (*net.TCPAddr, error) {
		return nil, assert.AnError
	}
	t.Cleanup(func() { resolveTCPAddr = originalResolver })

	log := logp.NewLogger("test")
	addresses := findLogonServer(false, log)
	require.Len(t, addresses, 1)
	assert.Equal(t, "ldap://DC02:389", addresses[0])
}

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
			name:      "Keep multi-level domain when skipping host",
			address:   "ldaps://corp.eu.example.com",
			expectDN:  "dc=eu,dc=example,dc=com",
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
		{
			name:      "Normalizes case and trailing dots",
			address:   "LDAPS://CORP.EXAMPLE.COM.:636",
			expectDN:  "dc=example,dc=com",
			expectErr: false,
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
