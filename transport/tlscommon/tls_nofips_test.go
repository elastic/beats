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

package tlscommon

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoFIPSCertificateAndKeys tests that encrypted private keys are supported in none FIPS mode
func TestNoFIPSCertificateAndKeys(t *testing.T) {
	t.Run("embed encrypted PKCS#1 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		password := "abcd1234"
		key, cert := makeKeyCertPair(t, blockTypePKCS1Encrypted, password)
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = cert
		cfg.Certificate.Key = key
		cfg.Certificate.Passphrase = password

		tlsC, err := LoadTLSConfig(cfg)
		require.NoError(t, err)
		assert.NotNil(t, tlsC)
	})

	t.Run("embed PKCS#8 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		password := "abcd1234"
		key, cert := makeKeyCertPair(t, blockTypePKCS8Encrypted, password)
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = cert
		cfg.Certificate.Key = key
		cfg.Certificate.Passphrase = password

		tlsC, err := LoadTLSConfig(cfg)
		require.NoError(t, err)
		assert.NotNil(t, tlsC)
	})
}

func TestEncryptedKeyPassphrase(t *testing.T) {
	const passphrase = "Abcd1234!" // passphrase for testdata/ca.encrypted.key
	t.Run("no passphrase", func(t *testing.T) {
		_, err := LoadTLSConfig(mustLoad(t, `
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.encrypted.key
    `))
		assert.ErrorContains(t, err, "no PEM blocks") // ReadPEMFile will generate an internal "no passphrase available" error that is logged and the no PEM blocks error is returned instead
	})

	t.Run("wrong passphrase", func(t *testing.T) {
		_, err := LoadTLSConfig(mustLoad(t, `
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.encrypted.key
    key_passphrase: "abcd1234!"
    `))
		assert.ErrorContains(t, err, "no PEM blocks") // ReadPEMFile will fail decrypting with x509.IncorrectPasswordError that will be logged and a no PEM blocks error is returned instead
	})

	t.Run("passphrase value", func(t *testing.T) {
		cfg, err := LoadTLSConfig(mustLoad(t, `
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.encrypted.key
    key_passphrase: Abcd1234!
    `))
		require.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Certificates), "expected 1 certificate to be loaded")
	})

	t.Run("passphrase file", func(t *testing.T) {
		fileName := writeTestFile(t, passphrase)
		cfg, err := LoadTLSConfig(mustLoad(t, fmt.Sprintf(`
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.encrypted.key
    key_passphrase_path: %s
    `, fileName)))
		require.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Certificates), "expected 1 certificate to be loaded")
	})

	t.Run("passphrase file empty", func(t *testing.T) {
		fileName := writeTestFile(t, "")
		_, err := LoadTLSConfig(mustLoad(t, fmt.Sprintf(`
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.encrypted.key
    key_passphrase_path: %s
    `, fileName)))
		assert.ErrorContains(t, err, "no PEM blocks") // ReadPEMFile will generate an internal "no passphrase available" error that is logged and the no PEM blocks error is returned instead
	})
}
