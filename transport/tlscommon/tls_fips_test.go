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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFIPSVerifyConnectionRejectsBadCerts verifies that all client verification
// modes reject a server certificate with a non-compliant key (1024-bit RSA,
// below the FIPS 140-3 minimum of 2048 bits).
func TestFIPSVerifyConnectionRejectsBadCerts(t *testing.T) {
	caCert, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	serverCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_invalid.crt"),
		filepath.Join("testdata", "fips_invalid.key"),
	)
	require.NoError(t, err)

	serverURL := startTestServer(t, "localhost:0", []tls.Certificate{serverCert})

	for _, mode := range []string{"full", "certificate", "strict", "none"} {
		t.Run("verification_mode="+mode, func(t *testing.T) {
			cfg, err := load(`enabled: true`)
			require.NoError(t, err)
			cfg.VerificationMode = tlsVerificationModes[mode]
			cfg.CAs = []string{string(caCert)}

			tlsCfg, err := LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err)

			err = dialTestServer(serverURL, tlsCfg)
			require.Error(t, err, "expected FIPS rejection for mode %q", mode)
			// cfg.CAs routes through CAReloader (CertificateReload.IsEnabled()=true by
			// default), so rootCAs is always dynamic. For all modes — including strict —
			// InsecureSkipVerify=true and checkPeerCertsFIPS fires explicitly
			// (golang/go#80074 means Go's own check is skipped with InsecureSkipVerify=true).
			// The exact error message is validated in TestFIPSVerifyConnectionCallbackRejectsBadCerts;
			// here we only assert rejection, because Go's TLS layer may report the error before
			// our VerifyConnection callback fires in some requirefips configurations.
		})
	}
}

// TestFIPSVerifyConnectionAllowsGoodCerts tests that compliant certificates
// (2048-bit RSA, testdata/fips_valid.crt) are accepted in all verification modes.
func TestFIPSVerifyConnectionAllowsGoodCerts(t *testing.T) {
	caCert, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	// fips_valid.crt is a 2048-bit RSA cert signed by the testdata CA — FIPS 140-3 compliant.
	serverCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_valid.crt"),
		filepath.Join("testdata", "fips_valid.key"),
	)
	require.NoError(t, err)

	serverURL := startTestServer(t, "localhost:0", []tls.Certificate{serverCert})

	for _, mode := range []string{"full", "certificate", "strict", "none"} {
		t.Run("verification_mode="+mode, func(t *testing.T) {
			cfg, err := load(`enabled: true`)
			require.NoError(t, err)
			cfg.VerificationMode = tlsVerificationModes[mode]
			cfg.CAs = []string{string(caCert)}

			tlsCfg, err := LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err)

			err = dialTestServer(serverURL, tlsCfg)
			require.NoError(t, err, "FIPS-compliant cert should be accepted for mode %q", mode)
		})
	}
}

// dialTestServer makes a single HTTPS GET to serverURL using the given TLSConfig
// and returns any TLS-level error (connection or handshake).
func dialTestServer(serverURL url.URL, cfg *TLSConfig) error {
	tlsNativeCfg := cfg.BuildModuleClientConfig(serverURL.Hostname())
	transport := &http.Transport{TLSClientConfig: tlsNativeCfg}
	transport.ForceAttemptHTTP2 = false
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	resp, err := client.Get(serverURL.String()) //nolint:noctx // testing
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// TestFIPSVerifyConnectionStaticCAsEndToEnd is an end-to-end TLS handshake
// companion to TestFIPSVerifyConnectionStaticCAs. It exercises the VerifyStrict
// static-CA client path through a real connection (InsecureSkipVerify=false),
// which is the code path that TestFIPSVerifyConnectionRejectsBadCerts does not
// cover (that test always uses CAReloader → dynamic pool → InsecureSkipVerify=true).
// For this path FIPS enforcement may come from Go's own check or from
// checkPeerCertsFIPS in the VerifyConnection callback; either way, non-compliant
// certs must be rejected and compliant ones accepted.
func TestFIPSVerifyConnectionStaticCAsEndToEnd(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	invalidServerCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_invalid.crt"),
		filepath.Join("testdata", "fips_invalid.key"),
	)
	require.NoError(t, err)

	validServerCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_valid.crt"),
		filepath.Join("testdata", "fips_valid.key"),
	)
	require.NoError(t, err)

	for _, tc := range []struct {
		name        string
		serverCert  tls.Certificate
		expectError bool
	}{
		{"non-FIPS cert rejected", invalidServerCert, true},
		{"FIPS-compliant cert accepted", validServerCert, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverURL := startTestServer(t, "127.0.0.1:0", []tls.Certificate{tc.serverCert})

			// Use a static CA pool: rootCAs.IsDynamic()==false → InsecureSkipVerify=false
			// in the resulting tls.Config. LoadTLSConfig always uses CAReloader (dynamic),
			// so we construct TLSConfig directly to reach this code path.
			cfg := &TLSConfig{
				Verification: VerifyStrict,
				rootCAs:      newStaticCertPool(pool),
				ServerName:   serverURL.Hostname(),
				Logger:       logptest.NewTestingLogger(t, ""),
			}

			err := dialTestServer(serverURL, cfg)
			if tc.expectError {
				require.Error(t, err, "non-FIPS cert must be rejected")
			} else {
				require.NoError(t, err, "FIPS-compliant cert must be accepted")
			}
		})
	}
}

// TestFIPSVerifyConnectionStaticCAs exercises the VerifyStrict static-CA
// client-side path. LoadTLSConfig always creates a dynamic CA pool via
// CAReloader, so this path is only reachable by constructing TLSConfig
// directly with a staticCertPool (rootCAs.IsDynamic()==false, no CASha256).
func TestFIPSVerifyConnectionStaticCAs(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	invalidCert := loadX509Cert(t, "fips_invalid.crt")
	validCert := loadX509Cert(t, "fips_valid.crt")

	cfg := &TLSConfig{
		Verification: VerifyStrict,
		rootCAs:      newStaticCertPool(pool),
		ServerName:   "localhost",
	}
	verifier := makeVerifyConnection(cfg, logptest.NewTestingLogger(t, ""))
	require.NotNil(t, verifier)

	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{invalidCert},
		VerifiedChains:   [][]*x509.Certificate{{invalidCert}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed by FIPS 140-3")

	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{validCert},
		VerifiedChains:   [][]*x509.Certificate{{validCert}},
	})
	require.NoError(t, err)
}

// TestFIPSNonFIPSIntermediateInVerifiedChain confirms that a non-FIPS intermediate
// (or root) in the verified chain is rejected even when the leaf is FIPS-compliant.
// checkAllChainsFIPS checks every cert in every chain, so a non-FIPS intermediate
// is caught regardless of its position.
func TestFIPSNonFIPSIntermediateInVerifiedChain(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	validCert := loadX509Cert(t, "fips_valid.crt")
	invalidCert := loadX509Cert(t, "fips_invalid.crt")

	t.Run("client static-bare: FIPS leaf + non-FIPS intermediate", func(t *testing.T) {
		cfg := &TLSConfig{
			Verification: VerifyStrict,
			rootCAs:      newStaticCertPool(pool),
			ServerName:   "localhost",
		}
		verifier := makeVerifyConnection(cfg, logptest.NewTestingLogger(t, ""))
		require.NotNil(t, verifier)

		err := verifier(tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{validCert},
			VerifiedChains:   [][]*x509.Certificate{{validCert, invalidCert}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
			"non-FIPS intermediate must be caught even when leaf is compliant")
	})

	t.Run("server static-bare: FIPS leaf + non-FIPS intermediate", func(t *testing.T) {
		cfg := &TLSConfig{
			Verification: VerifyStrict,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			clientCAs:    newStaticCertPool(pool),
		}
		verifier := makeVerifyServerConnection(cfg)
		require.NotNil(t, verifier)

		err := verifier(tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{validCert},
			VerifiedChains:   [][]*x509.Certificate{{validCert, invalidCert}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
			"non-FIPS intermediate must be caught even when leaf is compliant")
	})
}

// TestFIPSVerifyConnectionStaticCAsWithCASha256 tests the VerifyStrict + static
// CAs + CASha256 client code path. Go's stdlib handles chain and hostname
// (InsecureSkipVerify=false); our VerifyConnection callback runs verifyCAPin
// then checkPeerCertsFIPS. This exercises the len(CASha256)>0 branch in
// makeVerifyConnection for VerifyStrict with a static (non-dynamic) root pool.
func TestFIPSVerifyConnectionStaticCAsWithCASha256(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	caCert := loadX509Cert(t, "ca.crt")
	invalidCert := loadX509Cert(t, "fips_invalid.crt")
	validCert := loadX509Cert(t, "fips_valid.crt")

	cfg := &TLSConfig{
		Verification: VerifyStrict,
		rootCAs:      newStaticCertPool(pool),
		CASha256:     []string{Fingerprint(caCert)},
		ServerName:   "localhost",
	}
	verifier := makeVerifyConnection(cfg, logptest.NewTestingLogger(t, ""))
	require.NotNil(t, verifier)

	// verifyCAPin finds caCert in the chain and passes; checkPeerCertsFIPS then
	// rejects the non-FIPS leaf.
	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{invalidCert},
		VerifiedChains:   [][]*x509.Certificate{{invalidCert, caCert}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
		"VerifyStrict+CASha256 must reject non-FIPS cert after CA pin passes")

	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{validCert},
		VerifiedChains:   [][]*x509.Certificate{{validCert, caCert}},
	})
	require.NoError(t, err, "VerifyStrict+CASha256 must accept FIPS-compliant cert")
}

// TestFIPSVerifyServerConnectionRejectsBadCerts verifies that
// makeVerifyServerConnection enforces FIPS key-type constraints for every
// verification mode. For VerifyFull/VerifyCertificate the chain is verified
// first (using the testdata CA), then checkPeerCertsFIPS fires. For VerifyNone
// and VerifyStrict (static CAs, no CASha256) checkPeerCertsFIPS is the only
// check. Both reject (invalid cert) and accept (valid cert) cases are exercised.
func TestFIPSVerifyServerConnectionRejectsBadCerts(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	invalidCert := loadX509Cert(t, "fips_invalid.crt")
	validCert := loadX509Cert(t, "fips_valid.crt")

	for _, tc := range []struct {
		name string
		vm   TLSVerificationMode
	}{
		{"none", VerifyNone},
		{"full", VerifyFull},
		{"certificate", VerifyCertificate},
		{"strict", VerifyStrict},
	} {
		t.Run("verification_mode="+tc.name, func(t *testing.T) {
			cfg := &TLSConfig{
				Verification: tc.vm,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				clientCAs:    newStaticCertPool(pool), // IsDynamic()==false
			}
			verifier := makeVerifyServerConnection(cfg)
			require.NotNil(t, verifier)

			err := verifier(tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{invalidCert},
				VerifiedChains:   [][]*x509.Certificate{{invalidCert}},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
				"mode %q must reject non-FIPS cert", tc.name)

			err = verifier(tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{validCert},
				VerifiedChains:   [][]*x509.Certificate{{validCert}},
			})
			require.NoError(t, err, "mode %q must accept FIPS-compliant cert", tc.name)
		})
	}
}

// TestFIPSVerifyServerConnectionDynamicCAs tests the VerifyStrict + dynamic
// clientCAs server code path (clientCAs.IsDynamic()==true). InsecureSkipVerify
// is true in this path so chain verification is done manually via
// verifyCertsWithOpts, then checkPeerCertsFIPS fires.
func TestFIPSVerifyServerConnectionDynamicCAs(t *testing.T) {
	reloader, err := NewCAReloader(
		[]string{filepath.Join("testdata", "ca.crt")},
		0,
	)
	require.NoError(t, err)
	require.True(t, reloader.IsDynamic())

	invalidCert := loadX509Cert(t, "fips_invalid.crt")
	validCert := loadX509Cert(t, "fips_valid.crt")

	cfg := &TLSConfig{
		Verification: VerifyStrict,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		clientCAs:    reloader,
	}
	verifier := makeVerifyServerConnection(cfg)
	require.NotNil(t, verifier)

	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{invalidCert},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
		"dynamic server VerifyStrict must reject non-FIPS client cert")

	err = verifier(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{validCert},
	})
	require.NoError(t, err, "dynamic server VerifyStrict must accept FIPS-compliant client cert")
}

// TestFIPSVerifyConnectionCallbackRejectsBadCerts calls makeVerifyConnection
// directly for every verification mode and asserts that the callback returns
// the specific "not allowed by FIPS 140-3" error for a non-FIPS peer cert and
// nil for a FIPS-compliant cert.  This complements TestFIPSVerifyConnectionRejectsBadCerts
// (end-to-end), which cannot assert the exact error source because Go's TLS
// layer may reject the RSA-1024 server key before our VerifyConnection fires.
func TestFIPSVerifyConnectionCallbackRejectsBadCerts(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	invalidCert := loadX509Cert(t, "fips_invalid.crt")
	validCert := loadX509Cert(t, "fips_valid.crt")
	logger := logptest.NewTestingLogger(t, "")

	for _, tc := range []struct {
		name string
		vm   TLSVerificationMode
	}{
		{"none", VerifyNone},
		{"full", VerifyFull},
		{"certificate", VerifyCertificate},
		{"strict", VerifyStrict},
	} {
		t.Run("mode="+tc.name, func(t *testing.T) {
			cfg := &TLSConfig{
				Verification: tc.vm,
				rootCAs:      newStaticCertPool(pool),
				ServerName:   "localhost",
			}
			verifier := makeVerifyConnection(cfg, logger)
			require.NotNil(t, verifier)

			err := verifier(tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{invalidCert},
				VerifiedChains:   [][]*x509.Certificate{{invalidCert}},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
				"mode %q must report FIPS key-type error, got: %v", tc.name, err)

			err = verifier(tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{validCert},
				VerifiedChains:   [][]*x509.Certificate{{validCert}},
			})
			require.NoError(t, err, "mode %q must accept FIPS-compliant cert", tc.name)
		})
	}
}

// TestFIPSVerifyServerConnectionEndToEnd performs a real mTLS handshake to
// verify that BuildServerConfig wires makeVerifyServerConnection correctly and
// that the server rejects a client presenting a non-FIPS cert.
func TestFIPSVerifyServerConnectionEndToEnd(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	// Server presents a FIPS-compliant cert so the handshake is not blocked
	// by the server's own key type.
	serverCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_valid.crt"),
		filepath.Join("testdata", "fips_valid.key"),
	)
	require.NoError(t, err)

	invalidClientCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_invalid.crt"),
		filepath.Join("testdata", "fips_invalid.key"),
	)
	require.NoError(t, err)

	validClientCert, err := tls.LoadX509KeyPair(
		filepath.Join("testdata", "fips_valid.crt"),
		filepath.Join("testdata", "fips_valid.key"),
	)
	require.NoError(t, err)

	for _, tc := range []struct {
		name        string
		clientCert  tls.Certificate
		expectError bool
	}{
		{"non-FIPS client cert rejected", invalidClientCert, true},
		{"FIPS-compliant client cert accepted", validClientCert, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverTLSCfg := &TLSConfig{
				Verification: VerifyFull,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				clientCAs:    newStaticCertPool(pool),
				Certificates: []tls.Certificate{serverCert},
			}
			serverNativeCfg := serverTLSCfg.BuildServerConfig("")

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			var wg sync.WaitGroup
			wg.Go(func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					tlsConn := tls.Server(conn, serverNativeCfg)
					if err := tlsConn.Handshake(); err == nil {
						// Write a sentinel byte so the client can distinguish
						// acceptance from a TLS 1.3 rejection that only arrives
						// on the first application-data read.
						_, _ = tlsConn.Write([]byte{0})
					}
					tlsConn.Close()
				}
			})
			t.Cleanup(func() {
				ln.Close()
				wg.Wait()
			})

			clientCfg := &tls.Config{ //nolint:gosec // test: skip server cert verify, testing server-side FIPS enforcement only
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{tc.clientCert},
			}
			conn, dialErr := tls.Dial("tcp", ln.Addr().String(), clientCfg)
			var connErr error
			if dialErr == nil {
				// Under TLS 1.3 the client considers the handshake complete
				// before the server has validated the client cert. The server's
				// rejection alert (or sentinel byte on success) only arrives on
				// the first application-data read.
				var buf [1]byte
				_, connErr = conn.Read(buf[:])
				conn.Close()
			}

			err = dialErr
			if err == nil {
				err = connErr
			}

			if tc.expectError {
				require.Error(t, err, "server must reject non-FIPS client cert")
			} else {
				require.NoError(t, err, "server must accept FIPS-compliant client cert")
			}
		})
	}
}

// loadX509Cert parses a single PEM-encoded certificate from testdata.
func loadX509Cert(t *testing.T, filename string) *x509.Certificate {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	require.NoError(t, err)
	block, _ := pem.Decode(data)
	require.NotNil(t, block, "no PEM block in %s", filename)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

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

// TestFIPSMultipleVerifiedChainsAllMustPass verifies the weakest-link rule for
// multi-chain scenarios (e.g. a cross-signed cert with two valid paths to
// different roots). Every chain in cs.VerifiedChains must be FIPS-compliant;
// a single non-FIPS chain causes rejection even when another chain is clean.
func TestFIPSMultipleVerifiedChainsAllMustPass(t *testing.T) {
	caCertPEM, err := os.ReadFile(filepath.Join("testdata", "ca.crt"))
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caCertPEM))

	validCert := loadX509Cert(t, "fips_valid.crt")
	invalidCert := loadX509Cert(t, "fips_invalid.crt")

	cfg := &TLSConfig{
		Verification: VerifyStrict,
		rootCAs:      newStaticCertPool(pool),
		ServerName:   "localhost",
	}
	verifier := makeVerifyConnection(cfg, logptest.NewTestingLogger(t, ""))
	require.NotNil(t, verifier)

	t.Run("any non-FIPS chain causes rejection", func(t *testing.T) {
		// chain 0 is FIPS-clean; chain 1 has a non-FIPS intermediate.
		// The non-FIPS chain must cause rejection (weakest-link principle).
		err := verifier(tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{validCert},
			VerifiedChains: [][]*x509.Certificate{
				{validCert},
				{validCert, invalidCert},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
			"a non-FIPS chain must cause rejection even when another chain is FIPS-compliant")
	})

	t.Run("all-FIPS chains accepted", func(t *testing.T) {
		err := verifier(tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{validCert},
			VerifiedChains: [][]*x509.Certificate{
				{validCert},
				{validCert, validCert},
			},
		})
		require.NoError(t, err, "all-FIPS chains must be accepted")
	})
}

// TestFIPSCheckPeerCertsFIPSChain confirms that checkPeerCertsFIPS inspects
// every cert in the slice — not just the leaf — so a non-FIPS intermediate is
// caught even when the leaf is FIPS-compliant.
func TestFIPSCheckPeerCertsFIPSChain(t *testing.T) {
	validCert := loadX509Cert(t, "fips_valid.crt")
	invalidCert := loadX509Cert(t, "fips_invalid.crt")

	// FIPS-valid leaf followed by non-FIPS intermediate → rejected.
	err := checkPeerCertsFIPS([]*x509.Certificate{validCert, invalidCert})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed by FIPS 140-3",
		"non-FIPS intermediate must be caught even when leaf is compliant")

	// All-FIPS chain → accepted.
	err = checkPeerCertsFIPS([]*x509.Certificate{validCert, validCert})
	require.NoError(t, err, "all-FIPS chain must be accepted")
}
