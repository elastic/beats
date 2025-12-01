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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSearchFilter(t *testing.T) {
	validGUID := "{7fb125ee-ceaf-48ff-8385-32c516ab10ed}"
	guidBytes, _ := guidToBytes(validGUID)
	expectedEscaped := escapeBinaryForLDAP(guidBytes)

	boolTrue := true
	boolFalse := false

	tests := []struct {
		name              string
		ldapSearchAttr    string
		adGuidTranslation *bool
		isAD              bool
		input             string
		expect            string
		expectErr         bool
	}{
		{
			name:           "Default nil pointer converts when AD + objectGUID",
			ldapSearchAttr: "objectGUID",
			isAD:           true,
			input:          validGUID,
			expect:         expectedEscaped,
		},
		{
			name:           "Default nil pointer does not convert when non-AD",
			ldapSearchAttr: "objectGUID",
			isAD:           false,
			input:          validGUID,
			expect:         validGUID,
		},
		{
			name:           "Default nil pointer does not convert other attribute",
			ldapSearchAttr: "uid",
			isAD:           true,
			input:          validGUID,
			expect:         validGUID,
		},
		{
			name:              "Explicit true converts even if attribute different",
			ldapSearchAttr:    "uid",
			adGuidTranslation: &boolTrue,
			isAD:              false,
			input:             validGUID,
			expect:            expectedEscaped,
		},
		{
			name:              "Explicit false never converts",
			ldapSearchAttr:    "objectGUID",
			adGuidTranslation: &boolFalse,
			isAD:              true,
			input:             validGUID,
			expect:            validGUID,
		},
		{
			name:           "Invalid GUID with conversion attempt returns error",
			ldapSearchAttr: "objectGUID",
			isAD:           true,
			input:          "invalid-guid",
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processor{
				config: config{
					LDAPSearchAttribute: tt.ldapSearchAttr,
					ADGUIDTranslation:   tt.adGuidTranslation,
				},
				client: &ldapClient{isActiveDirectory: tt.isAD},
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
