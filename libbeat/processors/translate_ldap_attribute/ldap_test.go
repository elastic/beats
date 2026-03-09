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

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSearchFilter(t *testing.T) {
	validGUID := "{7fb125ee-ceaf-48ff-8385-32c516ab10ed}"
	guidBytes, _ := guidToBytes(validGUID)
	expectedEscaped := escapeBinaryForLDAP(guidBytes)

	tests := []struct {
		name              string
		ldapSearchAttr    string
		adGuidTranslation string
		input             string
		expect            string
		expectErr         bool
	}{
		{
			name:           "Auto mode converts when attribute is objectGUID",
			ldapSearchAttr: "objectGUID",
			input:          validGUID,
			expect:         expectedEscaped,
		},
		{
			name:           "Auto mode is case-insensitive",
			ldapSearchAttr: "objectguid",
			input:          validGUID,
			expect:         expectedEscaped,
		},
		{
			name:           "Auto mode does not convert other attribute",
			ldapSearchAttr: "uid",
			input:          validGUID,
			expect:         validGUID,
		},
		{
			name:              "Explicit true converts even if attribute different",
			ldapSearchAttr:    "uid",
			adGuidTranslation: guidTranslationAlways,
			input:             validGUID,
			expect:            expectedEscaped,
		},
		{
			name:              "Explicit false never converts",
			ldapSearchAttr:    "objectGUID",
			adGuidTranslation: guidTranslationNever,
			input:             validGUID,
			expect:            validGUID,
		},
		{
			name:           "Invalid GUID with conversion attempt returns error",
			ldapSearchAttr: "objectGUID",
			input:          "invalid-guid",
			expectErr:      true,
		},
		{
			name:           "Escapes filter characters when not converting",
			ldapSearchAttr: "uid",
			input:          "value*)(|(cn=*)",
			expect:         ldap.EscapeFilter("value*)(|(cn=*)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processor{
				config: config{
					LDAPSearchAttribute: tt.ldapSearchAttr,
					ADGUIDTranslation:   tt.adGuidTranslation,
				},
				client: &ldapClient{},
			}
			out, err := p.prepareSearchFilter(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expect, out)
		})
	}
}

func TestMaybeConvertMappedGUID(t *testing.T) {
	guidStr := "7fb125ee-ceaf-48ff-8385-32c516ab10ed"
	raw, err := guidToBytes(guidStr)
	require.NoError(t, err)

	tests := []struct {
		name      string
		cfg       config
		values    []string
		expect    []string
		expectErr bool
	}{
		{
			name: "auto converts objectGUID",
			cfg: config{
				LDAPMappedAttribute: "objectGUID",
				ADGUIDTranslation:   guidTranslationAuto,
			},
			values: []string{string(raw)},
			expect: []string{guidStr},
		},
		{
			name: "never leaves binary untouched",
			cfg: config{
				LDAPMappedAttribute: "objectGUID",
				ADGUIDTranslation:   guidTranslationNever,
			},
			values: []string{string(raw)},
			expect: []string{string(raw)},
		},
		{
			name: "non objectGUID attribute is ignored",
			cfg: config{
				LDAPMappedAttribute: "cn",
			},
			values: []string{string(raw)},
			expect: []string{string(raw)},
		},
		{
			name: "invalid length returns error",
			cfg: config{
				LDAPMappedAttribute: "objectGUID",
			},
			values:    []string{"short"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processor{config: tt.cfg}
			converted, err := p.maybeConvertMappedGUID(tt.values)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expect, converted)
		})
	}
}
