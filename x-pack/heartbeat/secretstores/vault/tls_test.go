// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// newTLSFakeVault returns a fake Vault served over TLS, with a certificate
// signed by its own CA, and that CA in PEM.
func newTLSFakeVault(t *testing.T) (*fakeVault, string) {
	t.Helper()
	fv := newFakeVault(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(fv.handle))
	t.Cleanup(srv.Close)
	fv.address = srv.URL
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	return fv, string(ca)
}

func TestTLS(t *testing.T) {
	fv, ca := newTLSFakeVault(t)
	tests := map[string]struct {
		ssl     map[string]any
		wantErr string
	}{
		"untrusted certificate":    {wantErr: "certificate"},
		"trusted CA":               {ssl: map[string]any{"certificate_authorities": []any{ca}}},
		"verification disabled":    {ssl: map[string]any{"verification_mode": "none"}},
		"other CA is not accepted": {ssl: map[string]any{"certificate_authorities": []any{otherCA(t)}}, wantErr: "certificate"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			entry := tokenEntry(fv.address)
			if tc.ssl != nil {
				entry["ssl"] = tc.ssl
			}
			s, _, _ := newTestStore(t, entry, nil)

			v, err := s.resolve(context.Background(), "myapp/creds#password")
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "the connection must be rejected")
				return
			}
			require.NoError(t, err, "the secret must be read over TLS")
			assert.Equal(t, testPassword, v, "unexpected secret value")
		})
	}
}

// otherCA returns a self-signed CA in PEM. It can't come from another
// httptest server: they all use the same certificate.
func otherCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "key must be generated")
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "other test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err, "certificate must be created")
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestClientTLSConfig(t *testing.T) {
	const caFile = "testdata/does-not-exist.pem"
	tests := map[string]struct {
		ssl            map[string]any
		wantMinVersion uint16
		wantErr        string
	}{
		"without ssl":                     {wantMinVersion: tls.VersionTLS12},
		"ssl without supported_protocols": {ssl: map[string]any{"verification_mode": "full"}, wantMinVersion: tls.VersionTLS12},
		"ssl with supported_protocols":    {ssl: map[string]any{"supported_protocols": []any{"TLSv1.3"}}, wantMinVersion: tls.VersionTLS13},
		"disabled ssl":                    {ssl: map[string]any{"enabled": false, "verification_mode": "none"}, wantMinVersion: tls.VersionTLS12},
		"missing CA file":                 {ssl: map[string]any{"certificate_authorities": []any{caFile}}, wantErr: "invalid 'ssl'"},
		"invalid verification mode":       {ssl: map[string]any{"verification_mode": "sometimes"}, wantErr: "verification_mode"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			entry := tokenEntry("https://vault.example.com:8200")
			if tc.ssl != nil {
				entry["ssl"] = tc.ssl
			}
			cfg, err := conf.NewConfigFrom(entry)
			require.NoError(t, err, "test entry must be valid")
			c := defaultConfig()
			err = cfg.Unpack(&c)
			if err == nil {
				err = c.validate()
			}
			var tlsConfig *tls.Config
			if err == nil {
				tlsConfig, err = clientTLSConfig(c, logptest.NewTestingLogger(t, ""))
			}
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "the ssl settings must be rejected")
				return
			}
			require.NoError(t, err, "the ssl settings must be valid")
			assert.Equal(t, tc.wantMinVersion, tlsConfig.MinVersion, "unexpected minimum TLS version")
			if tc.ssl != nil && tc.ssl["enabled"] != false {
				assert.Equal(t, "vault.example.com", tlsConfig.ServerName, "the server name must be the address host")
			}
		})
	}
}
