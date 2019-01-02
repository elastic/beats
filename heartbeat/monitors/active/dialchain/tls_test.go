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

package dialchain

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func Test_addCertMetdata(t *testing.T) {
	goodNotBefore := time.Now().Add(-time.Hour)
	goodNotAfter := time.Now().Add(time.Hour)
	goodCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             goodNotBefore,
		NotAfter:              goodNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	missingNotBeforeCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotAfter:              goodNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	missingNotAfterCert := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             goodNotBefore,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// notBefore is intentionally not a pointer type because go certificates don't have nullable time types
	// we cheat a bit and make not after nullable because there's no valid reason to create a cert with go's zero
	// time.
	// see the addCertMetadata function for more info on this.
	type expected struct {
		notBefore time.Time
		notAfter  *time.Time
	}
	tests := []struct {
		name     string
		chains   [][]*x509.Certificate
		expected expected
	}{
		{
			"Valid cert",
			[][]*x509.Certificate{{&goodCert}},
			expected{
				notBefore: goodNotBefore,
				notAfter:  &goodNotAfter,
			},
		},
		{
			"Missing not before",
			[][]*x509.Certificate{{&missingNotBeforeCert}},
			expected{
				notAfter: &goodNotAfter,
			},
		},
		{
			"Missing not after",
			[][]*x509.Certificate{{&missingNotAfterCert}},
			expected{
				notBefore: goodNotBefore,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := common.MapStr{}
			addCertMetdata(event, tt.chains)
			v, err := event.GetValue("tls.certificate_not_valid_before")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.notBefore, v)

			if tt.expected.notAfter != nil {
				v, err := event.GetValue("tls.certificate_not_valid_after")
				assert.NoError(t, err)
				assert.Equal(t, *tt.expected.notAfter, v)
			} else {
				ok, _ := event.HasKey("tls.certificate_not_valid_after")
				assert.False(t, ok, "event should not have not after %v", event)
			}
		})
	}
}
