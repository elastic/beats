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

// writeKeyAndCertFiles generates a cert/key pair and writes them to dir,
// returning the file paths. Each call generates a distinct pair.
func writeKeyAndCertFiles(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()

	keyPEM, certPEM := makeKeyCertPair(t, blockTypePKCS8, "")

	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, []byte(certPEM), 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte(keyPEM), 0o600))

	return certPath, keyPath
}

func TestNewCertReloader_ValidCertPair(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(t, dir)

	r, err := NewCertReloader(certPath, keyPath)
	require.NoError(t, err)

	got, err := r.GetCertificate(nil)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.NotEmpty(t, got.Certificate)
}

func TestNewCertReloader_InvalidCertPair(t *testing.T) {
	dir := t.TempDir()

	// Write a cert from one pair and a key from another — they won't match.
	_, certPEM1 := makeKeyCertPair(t, blockTypePKCS8, "")
	keyPEM2, _ := makeKeyCertPair(t, blockTypePKCS8, "")

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, []byte(certPEM1), 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte(keyPEM2), 0o600))

	_, err := NewCertReloader(certPath, keyPath)
	assert.Error(t, err)
}

func TestNewCertReloader_MissingFiles(t *testing.T) {
	_, err := NewCertReloader("/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err)
}

func TestNewCertReloader_EmptyPaths(t *testing.T) {
	_, err := NewCertReloader("", "")
	assert.Error(t, err)
}

func TestCertReloader_ReloadsAfterInterval(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(t, dir)

	r, err := NewCertReloader(certPath, keyPath, WithReloadInterval(100*time.Millisecond))
	require.NoError(t, err)

	// Capture the initial certificate bytes.
	initial, err := r.GetCertificate(nil)
	require.NoError(t, err)
	initialRaw := initial.Certificate[0]

	// Overwrite with a new cert/key pair (distinct from the original).
	writeKeyAndCertFiles(t, dir)

	// After the reload interval, GetCertificate should return the new cert.
	require.Eventually(t, func() bool {
		got, err := r.GetCertificate(nil)
		return err == nil && !bytes.Equal(got.Certificate[0], initialRaw)
	}, 2*time.Second, 50*time.Millisecond, "cert should have been reloaded")
}

func TestCertReloader_InvalidNewCert_KeepsOld(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(t, dir)

	r, err := NewCertReloader(certPath, keyPath, WithReloadInterval(100*time.Millisecond))
	require.NoError(t, err)

	initial, err := r.GetCertificate(nil)
	require.NoError(t, err)
	initialRaw := initial.Certificate[0]

	// Overwrite the cert file with invalid data.
	require.NoError(t, os.WriteFile(certPath, []byte("not a cert"), 0o600))

	// The original cert should remain served even after the reload interval,
	// because the new cert is invalid and the reload is silently skipped.
	require.Never(t, func() bool {
		got, err := r.GetCertificate(nil)
		if err != nil {
			return true
		}
		return !bytes.Equal(got.Certificate[0], initialRaw)
	}, 500*time.Millisecond, 50*time.Millisecond, "cert should not have changed after invalid reload")
}

func TestCertReloader_NoReloadBeforeInterval(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(t, dir)

	// Use a long reload interval so it won't elapse during the test.
	r, err := NewCertReloader(certPath, keyPath, WithReloadInterval(1*time.Hour))
	require.NoError(t, err)

	initial, err := r.GetCertificate(nil)
	require.NoError(t, err)
	initialRaw := initial.Certificate[0]

	// Replace the cert files on disk.
	writeKeyAndCertFiles(t, dir)

	// GetCertificate should still return the original cert because the
	// reload interval hasn't elapsed.
	got, err := r.GetCertificate(nil)
	require.NoError(t, err)
	assert.Equal(t, initialRaw, got.Certificate[0])
}
