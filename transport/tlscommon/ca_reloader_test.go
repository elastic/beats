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
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

func writeCAFile(t *testing.T, dir, name string) string {
	t.Helper()
	_, _, pair, err := certutil.NewRootCA()
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, pair.Cert, 0o600))
	return path
}

func TestNewCAReloader_ValidCA(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 5*time.Second)
	require.NoError(t, err)

	pool := r.GetCertPool()
	require.NotNil(t, pool)
}

func TestNewCAReloader_InvalidCA(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a cert"), 0o600))

	_, err := NewCAReloader([]string{path}, 5*time.Second)
	assert.Error(t, err)
}

func TestNewCAReloader_MissingFile(t *testing.T) {
	_, err := NewCAReloader([]string{"/nonexistent/ca.pem"}, 5*time.Second)
	assert.Error(t, err)
}

func TestNewCAReloader_EmptyPaths(t *testing.T) {
	_, err := NewCAReloader([]string{}, 5*time.Second)
	assert.Error(t, err)
}

func TestCAReloader_ReloadsAfterInterval(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 100*time.Millisecond)
	require.NoError(t, err)

	initialPool := r.GetCertPool()
	require.NotNil(t, initialPool)

	// Overwrite with a new CA.
	writeCAFile(t, dir, "ca.pem")

	// After the reload interval, GetCertPool should return a pool built
	// from the new CA file.
	require.Eventually(t, func() bool {
		pool := r.GetCertPool()
		return !pool.Equal(initialPool)
	}, 2*time.Second, 50*time.Millisecond, "CA pool should have been reloaded")
}

func TestCAReloader_InvalidNewCA_KeepsOld(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 100*time.Millisecond)
	require.NoError(t, err)

	initialPool := r.GetCertPool()
	require.NotNil(t, initialPool)

	// Overwrite with invalid data.
	require.NoError(t, os.WriteFile(caPath, []byte("not a cert"), 0o600))

	// The old pool should be preserved.
	require.Never(t, func() bool {
		pool := r.GetCertPool()
		return pool == nil
	}, 500*time.Millisecond, 50*time.Millisecond, "pool should not become nil after invalid reload")
}

func TestCAReloader_NoReloadBeforeInterval(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 1*time.Hour)
	require.NoError(t, err)

	initialPool := r.GetCertPool()

	// Replace the CA file.
	writeCAFile(t, dir, "ca.pem")

	// Should still return the original pool since interval hasn't elapsed.
	pool := r.GetCertPool()
	assert.True(t, pool.Equal(initialPool))
}

func TestCAReloader_PartialReloadFailure_UsesSuccessfulCAs(t *testing.T) {
	dir := t.TempDir()
	caPath1 := writeCAFile(t, dir, "ca1.pem")
	caPath2 := writeCAFile(t, dir, "ca2.pem")

	r, err := NewCAReloader([]string{caPath1, caPath2}, 100*time.Millisecond)
	require.NoError(t, err)

	initialPool := r.GetCertPool()
	require.NotNil(t, initialPool)

	// Corrupt one CA file; the other stays valid.
	require.NoError(t, os.WriteFile(caPath2, []byte("not a cert"), 0o600))

	// The pool should update to contain only the successful CA.
	require.Eventually(t, func() bool {
		pool := r.GetCertPool()
		return pool != nil && !pool.Equal(initialPool)
	}, 2*time.Second, 50*time.Millisecond,
		"pool should update with the CAs that loaded successfully")
}

func TestCAReloader_TotalReloadFailure_KeepsOldPool(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 100*time.Millisecond)
	require.NoError(t, err)

	initialPool := r.GetCertPool()
	require.NotNil(t, initialPool)

	// Corrupt the only CA file so all paths fail.
	require.NoError(t, os.WriteFile(caPath, []byte("not a cert"), 0o600))

	// The old pool should be preserved since all CAs failed.
	require.Never(t, func() bool {
		pool := r.GetCertPool()
		return !pool.Equal(initialPool)
	}, 500*time.Millisecond, 50*time.Millisecond,
		"pool should not change when all CAs fail to reload")
}

func TestCAReloader_AddTrustedCert_SurvivesReload(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 100*time.Millisecond)
	require.NoError(t, err)

	// Generate a separate CA cert and add it via AddTrustedCert.
	_, _, pair, err := certutil.NewRootCA()
	require.NoError(t, err)
	block, _ := pem.Decode(pair.Cert)
	require.NotNil(t, block)
	extraCA, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	r.AddTrustedCert(extraCA)

	// Verify it's in the pool now.
	pool := r.GetCertPool()
	_, err = extraCA.Verify(x509.VerifyOptions{Roots: pool})
	require.NoError(t, err, "trusted fingerprint cert should be in the pool before reload")

	// Trigger a reload by writing a new CA to the file and waiting.
	writeCAFile(t, dir, "ca.pem")
	require.Eventually(t, func() bool {
		pool = r.GetCertPool()
		_, err = extraCA.Verify(x509.VerifyOptions{Roots: pool})
		return err == nil
	}, 2*time.Second, 50*time.Millisecond,
		"trusted fingerprint cert should survive CA reload")
}

func TestCAReloader_AddTrustedCert_Deduplicates(t *testing.T) {
	dir := t.TempDir()
	caPath := writeCAFile(t, dir, "ca.pem")

	r, err := NewCAReloader([]string{caPath}, 1*time.Hour)
	require.NoError(t, err)

	_, _, pair, err := certutil.NewRootCA()
	require.NoError(t, err)
	block, _ := pem.Decode(pair.Cert)
	require.NotNil(t, block)
	extraCA, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	r.AddTrustedCert(extraCA)
	r.AddTrustedCert(extraCA)
	r.AddTrustedCert(extraCA)

	r.mu.RLock()
	count := len(r.trustedFingerprintCerts)
	r.mu.RUnlock()
	assert.Equal(t, 1, count, "duplicate certs should not be added")
}

func TestCAReloader_InlinePEM(t *testing.T) {
	_, _, pair, err := certutil.NewRootCA()
	require.NoError(t, err)

	// Pass the PEM directly as an inline string (starts with "-").
	r, err := NewCAReloader([]string{string(pair.Cert)}, 5*time.Second)
	require.NoError(t, err)

	pool := r.GetCertPool()
	require.NotNil(t, pool)

	// Verify the pool contains the CA.
	block, _ := pem.Decode(pair.Cert)
	require.NotNil(t, block)
	ca, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	chains, err := ca.Verify(x509.VerifyOptions{Roots: pool})
	require.NoError(t, err)
	assert.NotEmpty(t, chains)
}
