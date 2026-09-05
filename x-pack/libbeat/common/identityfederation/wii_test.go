// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestResolveWIITokenEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		issuerURL   string
		expected    string
		expectedErr string
	}{
		{
			name:      "plain https URL",
			issuerURL: "https://workload-identity-issuer-internal.eu-west-1.aws.svc.elastic.cloud",
			expected:  "https://workload-identity-issuer-internal.eu-west-1.aws.svc.elastic.cloud/token",
		},
		{
			name:      "trailing slashes are stripped",
			issuerURL: "https://issuer.example.com//",
			expected:  "https://issuer.example.com/token",
		},
		{
			name:        "http scheme is rejected",
			issuerURL:   "http://issuer.example.com",
			expectedErr: "must use the https scheme",
		},
		{
			name:        "empty URL is rejected",
			issuerURL:   "",
			expectedErr: "must be configured",
		},
		{
			name:        "host is required",
			issuerURL:   "https:///token",
			expectedErr: "must include a host",
		},
		{
			name:        "query string is rejected",
			issuerURL:   "https://issuer.example.com?region=eu",
			expectedErr: "must not include a query string or fragment",
		},
		{
			name:        "URL ending in /token is rejected",
			issuerURL:   "https://issuer.example.com/token",
			expectedErr: "omit the /token path suffix",
		},
		{
			name:        "URL ending in /token/ is rejected",
			issuerURL:   "https://issuer.example.com/token/",
			expectedErr: "omit the /token path suffix",
		},
		{
			name:      "subpath without /token is accepted",
			issuerURL: "https://proxy.example.com/wii",
			expected:  "https://proxy.example.com/wii/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := resolveWIITokenEndpoint(tt.issuerURL)
			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, endpoint)
		})
	}
}

func TestParseWIITokenResponse(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()

	tests := []struct {
		name        string
		body        string
		expectedErr string
	}{
		{
			name: "valid envelope",
			body: fmt.Sprintf(`{"token":"eyJhbGciOiJSUzI1NiJ9.e30.sig","expires_at":%d}`, future),
		},
		{
			name: "unknown fields are tolerated",
			body: fmt.Sprintf(`{"token":"eyJx.y.z","expires_at":%d,"future_field":true}`, future),
		},
		{
			name:        "missing expires_at is rejected",
			body:        `{"token":"eyJx.y.z"}`,
			expectedErr: "missing the expires_at field",
		},
		{
			name:        "missing token is rejected",
			body:        fmt.Sprintf(`{"expires_at":%d}`, future),
			expectedErr: "missing the token field",
		},
		{
			name:        "raw JWT body is rejected",
			body:        "eyJhbGciOiJSUzI1NiJ9.e30.sig",
			expectedErr: "parsing WII token response",
		},
		{
			name:        "expired token is rejected",
			body:        fmt.Sprintf(`{"token":"eyJx.y.z","expires_at":%d}`, time.Now().Add(-time.Hour).Unix()),
			expectedErr: "is in the past",
		},
		{
			name:        "epoch-millis encoding bug is caught by the lifetime ceiling",
			body:        fmt.Sprintf(`{"token":"eyJx.y.z","expires_at":%d}`, time.Now().UnixMilli()),
			expectedErr: "exceeds the maximum acceptable lifetime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := parseWIITokenResponse([]byte(tt.body))
			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, token)
			require.Equal(t, future, expiresAt.Unix())
		})
	}
}

type wiiTestPKI struct {
	caPEMPath  string
	certPath   string
	keyPath    string
	serverCert tls.Certificate
	clientPool *x509.CertPool
}

func newWIITestPKI(t *testing.T) wiiTestPKI {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "wii-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	newLeaf := func(cn string, extUsage x509.ExtKeyUsage) (tls.Certificate, []byte, []byte) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		template := &x509.Certificate{
			SerialNumber: big.NewInt(time.Now().UnixNano()),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{extUsage},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		require.NoError(t, err)
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyDER, err := x509.MarshalECPrivateKey(key)
		require.NoError(t, err)
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		require.NoError(t, err)
		return cert, certPEM, keyPEM
	}

	serverCert, _, _ := newLeaf("wii-test-server", x509.ExtKeyUsageServerAuth)
	_, clientCertPEM, clientKeyPEM := newLeaf("wii-test-client", x509.ExtKeyUsageClientAuth)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caPath := filepath.Join(dir, "ca.crt")
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))
	require.NoError(t, os.WriteFile(certPath, clientCertPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, clientKeyPEM, 0o600))

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	return wiiTestPKI{
		caPEMPath:  caPath,
		certPath:   certPath,
		keyPath:    keyPath,
		serverCert: serverCert,
		clientPool: pool,
	}
}

func newWIITestServer(t *testing.T, pki wiiTestPKI, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{pki.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pki.clientPool,
		MinVersion:   tls.VersionTLS12,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func tokenResponse(token string, expiresAt time.Time) []byte {
	body, _ := json.Marshal(map[string]any{"token": token, "expires_at": expiresAt.Unix()})
	return body
}

func TestWIITokenSourceMTLSExchange(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/token", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NotEmpty(t, r.TLS.PeerCertificates, "client certificate must be presented")

		var req struct {
			Aud string `json:"aud"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "sts.amazonaws.com", req.Aud)

		_, _ = w.Write(tokenResponse("eyJtest.jwt.value", time.Now().Add(time.Hour)))
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	token, err := source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJtest.jwt.value"), token)

	token, err = source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJtest.jwt.value"), token)
	require.Equal(t, int32(1), requests.Load())
}

func TestWIITokenSourceRefreshesNearExpiry(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
		n := requests.Add(1)
		_, _ = w.Write(tokenResponse(fmt.Sprintf("eyJtoken-%d", n), time.Now().Add(wiiRefreshBeforeExpiry/2)))
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	token, err := source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJtoken-1"), token)

	token, err = source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJtoken-2"), token, "a token inside the refresh margin must be refetched")
}

func TestWIITokenSourceRetriesTransientErrors(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(tokenResponse("eyJrecovered", time.Now().Add(time.Hour)))
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	token, err := source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJrecovered"), token)
	require.Equal(t, int32(3), requests.Load(), "two 503s then success within the attempt budget")
}

func TestWIITokenSourceDoesNotRetryClientErrors(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	_, err = source.GetIdentityToken()
	require.ErrorContains(t, err, "HTTP 401")
	require.Equal(t, int32(1), requests.Load(), "4xx responses must not be retried")

	_, err = source.GetIdentityToken()
	require.Error(t, err)
	require.Equal(t, int32(2), requests.Load())
}

func TestWIITokenSourceServesCachedTokenOnRefreshFailure(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
		n := requests.Add(1)
		if n == 1 {
			_, _ = w.Write(tokenResponse("eyJcached-token", time.Now().Add(wiiRefreshBeforeExpiry/2)))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	token, err := source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJcached-token"), token)

	// Refresh attempt fails (503), but cached token has not hard-expired yet — must not error.
	token, err = source.GetIdentityToken()
	require.NoError(t, err)
	require.Equal(t, []byte("eyJcached-token"), token, "cached token must be returned when refresh fails but token is still valid")
}

func TestWIITokenSourcePicksUpRotatedCert(t *testing.T) {
	pki := newWIITestPKI(t)
	t.Setenv(WIISSLCAFileEnvVar, pki.caPEMPath)

	var requests atomic.Int32
	srv := newWIITestServer(t, pki, func(w http.ResponseWriter, _ *http.Request) {
		n := requests.Add(1)
		_, _ = w.Write(tokenResponse(fmt.Sprintf("eyJtoken-%d", n), time.Now().Add(wiiRefreshBeforeExpiry/2)))
	})

	source, err := NewWIITokenSource(srv.URL, pki.certPath, pki.keyPath, "sts.amazonaws.com", logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	_, err = source.GetIdentityToken()
	require.NoError(t, err)

	// Rotate: overwrite the client cert/key in place with a fresh CA-signed pair
	// (the controller rotates the mounted Secret the same way).
	rotated := newWIITestPKI(t)
	// The rotated client cert is signed by a different CA the server does not trust,
	// so the next fetch must fail — proving the cert is re-read from disk per fetch.
	certPEM, err := os.ReadFile(rotated.certPath)
	require.NoError(t, err)
	keyPEM, err := os.ReadFile(rotated.keyPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(pki.certPath, certPEM, 0o600)) //nolint:gosec // G703: test-owned temp file
	require.NoError(t, os.WriteFile(pki.keyPath, keyPEM, 0o600))   //nolint:gosec // G703: test-owned temp file

	// Expire the cache so the stale-token fallback does not mask the TLS error.
	source.cachedExpiry = time.Now().Add(-time.Second)

	_, err = source.GetIdentityToken()
	require.Error(t, err, "a rotated (untrusted) cert must be presented on the next fetch")
}
