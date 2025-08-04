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
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFIPSCertifacteAndKeys tests that encrypted private keys fail in FIPS mode
func TestFIPSCertificateAndKeys(t *testing.T) {
	t.Run("embed encrypted PKCS#1 key", func(t *testing.T) {
		password := "abcd1234"

		keyFile, err := os.Open(filepath.Join("testdata", "key.pkcs1encrypted.pem"))
		require.NoError(t, err)
		defer keyFile.Close()
		rawKey, err := io.ReadAll(keyFile)
		require.NoError(t, err)

		certFile, err := os.Open(filepath.Join("testdata", "cert.pkcs1encrypted.pem"))
		require.NoError(t, err)
		defer certFile.Close()
		rawCert, err := io.ReadAll(certFile)
		require.NoError(t, err)

		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = string(rawCert)
		cfg.Certificate.Key = string(rawKey)
		cfg.Certificate.Passphrase = password

		_, err = LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
		require.Error(t, err)
		assert.ErrorIs(t, err, errors.ErrUnsupported, err)
	})

	t.Run("embed encrypted PKCS#8 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		password := "abcdefg1234567"
		key, cert := makeKeyCertPair(t, blockTypePKCS8Encrypted, password)
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = cert
		cfg.Certificate.Key = key
		cfg.Certificate.Passphrase = password

		_, err = LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
		assert.ErrorIs(t, err, errors.ErrUnsupported)
	})
}
