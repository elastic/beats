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
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// midBodyLine returns a line from the middle of a PEM block, i.e. real base64
// content (never a header/footer), so tests can assert it is never leaked.
func midBodyLine(t *testing.T, pemStr string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(pemStr), "\n")
	require.Greater(t, len(lines), 2, "PEM should have a header, body and footer")
	return lines[len(lines)/2]
}

// body returns everything after the first line of a PEM block, so tests can
// replace the "-----BEGIN ...-----" header with a malformed one while keeping
// the real (sensitive) body intact.
func body(pemStr string) string {
	return pemStr[strings.Index(pemStr, "\n")+1:]
}

// TestLoadCertificateDoesNotLeakPEM a malformed inline PEM (a private key in
// particular) must never appear in the error returned by LoadCertificate. The
// internal log.Errorf calls are built from the same safe inputs as the returned
// error, so a clean returned error implies a clean log line too.
func TestLoadCertificateDoesNotLeakPEM(t *testing.T) {
	keyPEM, certPEM := makeKeyCertPair(t, blockTypePKCS8, "")

	tests := []struct {
		name string
		// certificate/key are fed to LoadCertificate; secret is the sensitive
		// line that must not appear in the error; mustContain is a safe
		// substring the error is expected to carry (proving the redaction path
		// ran rather than a raw echo).
		certificate string
		key         string
		secret      string
		mustContain string
	}{
		{
			name:        "malformed key, header keeps leading dashes",
			certificate: certPEM,
			key:         "-----asdasd-----\n" + body(keyPEM),
			secret:      midBodyLine(t, keyPEM),
			// The key source is never echoed at all (not even a placeholder).
			mustContain: "failed reading key",
		},
		{
			name:        "malformed key, header without leading dashes",
			certificate: certPEM,
			key:         "asdasd-----\n" + body(keyPEM),
			secret:      midBodyLine(t, keyPEM),
			mustContain: "failed reading key",
		},
		{
			name:        "malformed cert, header keeps leading dashes",
			certificate: "-----asdasd-----\n" + body(certPEM),
			key:         keyPEM,
			secret:      midBodyLine(t, certPEM),
			mustContain: "PEM REDACTED",
		},
		{
			name:        "malformed cert, header without leading dashes",
			certificate: "asdasd-----\n" + body(certPEM),
			key:         keyPEM,
			secret:      midBodyLine(t, certPEM),
			mustContain: "PEM REDACTED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadCertificate(&CertificateConfig{
				Certificate: tc.certificate,
				Key:         tc.key,
			})
			require.Error(t, err)

			msg := err.Error()
			assert.NotContains(t, msg, tc.secret,
				"error leaks PEM body material:\n%s", msg)
			assert.NotContains(t, msg, "PRIVATE KEY",
				"error leaks a PEM private-key block:\n%s", msg)
			assert.NotContains(t, msg, "BEGIN",
				"error leaks PEM block content:\n%s", msg)
			assert.Contains(t, msg, tc.mustContain,
				"error should carry the safe label/message:\n%s", msg)
		})
	}
}

// TestLoadCertificateAuthoritiesDoesNotLeakPEM is the LoadCertificateAuthorities
// counterpart to TestLoadCertificateDoesNotLeakPEM: a malformed inline CA must
// never appear in the errors returned.
func TestLoadCertificateAuthoritiesDoesNotLeakPEM(t *testing.T) {
	_, certPEM := makeKeyCertPair(t, blockTypePKCS8, "")

	tests := []struct {
		name string
		ca   string
	}{
		{
			name: "malformed CA, header keeps leading dashes",
			ca:   "-----asdasd-----\n" + body(certPEM),
		},
		{
			name: "malformed CA, header without leading dashes",
			ca:   "asdasd-----\n" + body(certPEM),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret := midBodyLine(t, certPEM)

			_, errs := LoadCertificateAuthorities([]string{tc.ca})
			require.NotEmpty(t, errs)

			for _, err := range errs {
				msg := err.Error()
				assert.NotContains(t, msg, secret,
					"error leaks PEM body material:\n%s", msg)
				assert.NotContains(t, msg, "BEGIN",
					"error leaks PEM block content:\n%s", msg)
				assert.Contains(t, msg, "PEM REDACTED",
					"error should label the inline source as redacted")
			}
		})
	}
}

// TestLoadCertificateValidInlinePEM is a regression guard: a well-formed inline
// PEM pair must still load successfully after the redaction changes.
func TestLoadCertificateValidInlinePEM(t *testing.T) {
	keyPEM, certPEM := makeKeyCertPair(t, blockTypePKCS8, "")

	cert, err := LoadCertificate(&CertificateConfig{Certificate: certPEM, Key: keyPEM})
	require.NoError(t, err)
	require.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
}

// TestLoadCertificateMissingFileShowsPath is a regression guard: a genuine
// (single-line) file path that does not exist must still be reported verbatim
// so users can fix it — only inline content is redacted.
func TestLoadCertificateMissingFileShowsPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.pem")

	_, err := LoadCertificate(&CertificateConfig{Certificate: missing, Key: missing})
	require.Error(t, err)
	assert.Contains(t, err.Error(), missing, "missing file path should be visible")
	assert.NotContains(t, err.Error(), "PEM REDACTED", "a real path must not be redacted")
}

// TestNewPEMReaderMissingFileDropsPathFromError verifies that when os.Open
// fails, NewPEMReader returns an *fs.PathError with its Path cleared: a
// malformed inline PEM mistaken for a path must never be echoed, while callers
// type-asserting for *fs.PathError (or matching fs.ErrNotExist) still work.
func TestNewPEMReaderMissingFileDropsPathFromError(t *testing.T) {
	// A single-line, path-like string (not inline PEM) pointing at a
	// non-existent file, so os.Open fails with an *fs.PathError.
	missing := filepath.Join(t.TempDir(), "does-not-exist.pem")

	_, err := NewPEMReader(missing)
	require.Error(t, err)

	var pathErr *fs.PathError
	require.ErrorAs(t, err, &pathErr, "error type must remain *fs.PathError for callers matching on it")
	assert.Empty(t, pathErr.Path, "the Path component must be dropped so a mistaken inline PEM can never leak")
	assert.NotContains(t, err.Error(), missing, "the raw input must not appear in the error")
	assert.ErrorIs(t, err, fs.ErrNotExist, "fs.ErrNotExist matching must still work after dropping Path")
}

func TestIsInlinePEM(t *testing.T) {
	keyPEM, certPEM := makeKeyCertPair(t, blockTypePKCS8, "")

	tests := map[string]struct {
		in   string
		want bool
	}{
		"valid inline cert":                {in: certPEM, want: true},
		"valid inline key":                 {in: keyPEM, want: true},
		"dashless single-line base64 body": {in: strings.Repeat("A", minInlinePEMBodyLen), want: true},
		"malformed, leading dashes":        {in: "-----asdasd-----\n" + body(keyPEM), want: true},
		"malformed, multiline no dashes":   {in: "asdasd-----\n" + body(keyPEM), want: true},
		"single line with PEM armor":       {in: "asdasd-----MIIE-----END PRIVATE KEY-----", want: true},

		"empty string":       {in: "", want: false},
		"unix file path":     {in: "/etc/pki/tls/key.pem", want: false},
		"windows file path":  {in: `C:\pki\key.pem`, want: false},
		"relative file path": {in: "certs/key.pem", want: false},
		"path with trailing newline (YAML block scalar)":  {in: "/etc/pki/tls/key.pem\n", want: false},
		"path with surrounding whitespace":                {in: "  /etc/pki/tls/key.pem  \n", want: false},
		"long path without extension but with separators": {in: "/very/long/path/to/some/certificate/keyfile", want: false}, // shorter than any key
		"short bare base64-looking name":                  {in: "AAAABBBBCCCC", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isInlinePEM(tc.in))
		})
	}
}

func TestPemSource(t *testing.T) {
	keyPEM, _ := makeKeyCertPair(t, blockTypePKCS8, "")

	assert.Equal(t, "PEM REDACTED", pemSource(keyPEM))
	assert.Equal(t, "PEM REDACTED", pemSource("asdasd-----\nbody\n-----END PRIVATE KEY-----"))
	assert.Equal(t, "/etc/pki/key.pem", pemSource("/etc/pki/key.pem"))
	assert.Equal(t, "/etc/pki/key.pem\n", pemSource("/etc/pki/key.pem\n"),
		"a path with a trailing newline (e.g. YAML block scalar) must not be redacted")
}

// TestLoadCertificatePathWithTrailingNewlineIsNotRedacted a genuine file path
// with trailing whitespace/newline (as produced by a YAML block scalar like
// `cert: |`) must still be classified and reported as a path rather than
// misclassified as inline PEM content and redacted. The trailing newline is
// still part of the path handed to os.Open, so the file is not found either
// way -- this only guards that the real path stays visible in the resulting
// error rather than being swapped for a fabricated "PEM REDACTED". The OS
// error text for an invalid path differs by platform (e.g. Windows rejects
// an embedded newline outright instead of reporting "no such file"), so the
// assertion checks for the path substring rather than any particular error
// string.
func TestLoadCertificatePathWithTrailingNewlineIsNotRedacted(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(t, dir)

	_, err := LoadCertificate(&CertificateConfig{
		Certificate: certPath + "\n",
		Key:         keyPath + "\n",
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "PEM REDACTED",
		"a file path must not be redacted just because it has a trailing newline")
	assert.Contains(t, err.Error(), certPath,
		"the real path should still be visible in the error, proving it was attempted via os.Open rather than treated as inline content")
}
