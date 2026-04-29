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

// Encrypted private keys use legacy PEM encryption (MD5/DES) which is
// unavailable in FIPS mode.

package tlscommon

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeEncryptedKeyAndCertFiles(t *testing.T, dir string, blockType int, passphrase string) (certPath, keyPath string) {
	t.Helper()

	keyPEM, certPEM := makeKeyCertPair(t, blockType, passphrase)

	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, []byte(certPEM), 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte(keyPEM), 0o600))

	return certPath, keyPath
}

func TestNewCertReloader_WithPassphrase_PKCS1(t *testing.T) {
	dir := t.TempDir()
	passphrase := "test-passphrase"
	certPath, keyPath := writeEncryptedKeyAndCertFiles(t, dir, blockTypePKCS1Encrypted, passphrase)

	r, err := NewCertReloader(certPath, keyPath, WithPassphrase(passphrase))
	require.NoError(t, err)

	got, err := r.GetCertificate(nil)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.NotEmpty(t, got.Certificate)
}

func TestNewCertReloader_WithPassphrase_PKCS8(t *testing.T) {
	dir := t.TempDir()
	passphrase := "test-passphrase"
	certPath, keyPath := writeEncryptedKeyAndCertFiles(t, dir, blockTypePKCS8Encrypted, passphrase)

	r, err := NewCertReloader(certPath, keyPath, WithPassphrase(passphrase))
	require.NoError(t, err)

	got, err := r.GetCertificate(nil)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.NotEmpty(t, got.Certificate)
}

func TestNewCertReloader_WithPassphrase_WrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeEncryptedKeyAndCertFiles(t, dir, blockTypePKCS8Encrypted, "correct-passphrase")

	_, err := NewCertReloader(certPath, keyPath, WithPassphrase("wrong-passphrase"))
	assert.Error(t, err)
}

func TestCertReloader_WithPassphrase_ReloadsAfterInterval(t *testing.T) {
	dir := t.TempDir()
	passphrase := "test-passphrase"
	certPath, keyPath := writeEncryptedKeyAndCertFiles(t, dir, blockTypePKCS8Encrypted, passphrase)

	r, err := NewCertReloader(certPath, keyPath, WithPassphrase(passphrase), WithReloadInterval(100*time.Millisecond))
	require.NoError(t, err)

	initial, err := r.GetCertificate(nil)
	require.NoError(t, err)
	initialRaw := initial.Certificate[0]

	writeEncryptedKeyAndCertFiles(t, dir, blockTypePKCS8Encrypted, passphrase)

	require.Eventually(t, func() bool {
		got, err := r.GetCertificate(nil)
		return err == nil && !bytes.Equal(got.Certificate[0], initialRaw)
	}, 2*time.Second, 50*time.Millisecond, "cert should have been reloaded")
}
