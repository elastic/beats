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
	"github.com/xdg-go/scram"

	"github.com/elastic/sarama"

	"github.com/elastic/beats/v7/testing/testutils"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		mechanism string
		wantErr   bool
	}{
		// All accepted mechanisms, in both canonical and lowercase forms.
		{mechanism: ""},
		{mechanism: "PLAIN"},
		{mechanism: "plain"}, // exercises strings.ToUpper
		{mechanism: "SCRAM-SHA-256"},
		{mechanism: "scram-sha-256"}, // exercises strings.ToUpper
		{mechanism: "SCRAM-SHA-512"},
		{mechanism: "scram-sha-512"}, // exercises strings.ToUpper
		// OAUTHBEARER passes validation (bug tracked separately — it currently
		// falls through ConfigureSarama's default and authenticates as PLAIN).
		{mechanism: "OAUTHBEARER"},
		// Unsupported mechanisms.
		{mechanism: "GSSAPI", wantErr: true},
		{mechanism: "bogus", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.mechanism, func(t *testing.T) {
			cfg := SaslConfig{SaslMechanism: tc.mechanism}
			err := cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigureSarama(t *testing.T) {
	tests := []struct {
		mechanism        string
		wantHandshake    bool
		wantMechanism    string
		wantSCRAMFactory bool
	}{
		{
			mechanism:        "SCRAM-SHA-256",
			wantHandshake:    true,
			wantMechanism:    saslTypeSCRAMSHA256,
			wantSCRAMFactory: true,
		},
		{
			mechanism:        "SCRAM-SHA-512",
			wantHandshake:    true,
			wantMechanism:    saslTypeSCRAMSHA512,
			wantSCRAMFactory: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.mechanism, func(t *testing.T) {
			cfg := SaslConfig{SaslMechanism: tc.mechanism}
			saramaCfg := sarama.NewConfig()

			cfg.ConfigureSarama(saramaCfg)

			assert.Equal(t, tc.wantHandshake, saramaCfg.Net.SASL.Handshake)
			assert.Equal(t, tc.wantMechanism, string(saramaCfg.Net.SASL.Mechanism))
			if tc.wantSCRAMFactory {
				require.NotNil(t, saramaCfg.Net.SASL.SCRAMClientGeneratorFunc)
				client := saramaCfg.Net.SASL.SCRAMClientGeneratorFunc()
				assert.IsType(t, &XDGSCRAMClient{}, client)
			}
		})
	}
}

// TestSCRAMUsesValidatedPBKDF2 confirms that the FIPS-validated stdlib crypto/pbkdf2
// is linked, not the pure-Go github.com/xdg-go/pbkdf2. Under GODEBUG=fips140=only,
// stdlib crypto/pbkdf2 rejects salts shorter than 128 bits; xdg-go/pbkdf2 accepts
// any salt and cannot return an error at all. An error here is proof of the right
// implementation. The test skips itself outside fips140=only so it never produces a
// false-positive pass on a normal build.
func TestSCRAMUsesValidatedPBKDF2(t *testing.T) {
	testutils.SkipIfNotFIPSOnly(t, "guard only observable under GODEBUG=fips140=only")

	client, err := SHA256.NewClient("user", "password", "")
	require.NoError(t, err)

	// "tooshort" is 8 bytes, below the 128-bit (16-byte) floor enforced by
	// crypto/pbkdf2 in fips140=only mode. If this returns nil, the wrong
	// (non-validated) PBKDF2 implementation is linked.
	_, err = client.GetStoredCredentialsWithError(scram.KeyFactors{Salt: "tooshort", Iters: 4096})
	require.ErrorContains(t, err, "shorter than 128 bits")
}
