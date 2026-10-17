// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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
)

func newTLSFakeCCP(t *testing.T) (*fakeCCP, string) {
	t.Helper()
	fc := newFakeCCP(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(fc.handle))
	t.Cleanup(srv.Close)
	fc.address = srv.URL
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	return fc, string(ca)
}

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

func TestTLS(t *testing.T) {
	fc, ca := newTLSFakeCCP(t)
	tests := map[string]struct {
		ssl        map[string]any
		wantErr    string
		wantNoSpun bool
	}{
		"untrusted certificate":    {wantErr: "certificate", wantNoSpun: true},
		"trusted CA":               {ssl: map[string]any{"certificate_authorities": []any{ca}}},
		"verification disabled":    {ssl: map[string]any{"verification_mode": "none"}},
		"other CA is not accepted": {ssl: map[string]any{"certificate_authorities": []any{otherCA(t)}}, wantErr: "certificate", wantNoSpun: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			entry := ccpEntry(fc.address)
			if tc.ssl != nil {
				entry["ssl"] = tc.ssl
			}
			s, _, sleeps := newTestStore(t, entry, nil)

			v, err := s.resolve(context.Background(), "MySafe/MyAccount#password")
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "the connection must be rejected")
				if tc.wantNoSpun {
					assert.Empty(t, *sleeps, "certificate errors must not be retried")
				}
				return
			}
			require.NoError(t, err, "the secret must be read over TLS")
			assert.Equal(t, testPassword, v, "unexpected secret value")
		})
	}
}
