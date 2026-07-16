// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestAuthenticator(t *testing.T) {
	// Pre-generate CA and certificates for the TLS test case
	caCert, err := tlscommontest.GenCA()
	require.NoError(t, err)

	serverCerts, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "", []string{}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err)

	testCases := []struct {
		name                     string
		setupConfig              func(t *testing.T) *Config
		continueOnError          bool
		skipStart                bool
		expectStartError         bool
		expectStatus             componentstatus.Status
		expectHTTPClientType     string // "httpClientProvider" or "errorRoundTripperProvider"
		testHTTPRequest          bool
		testRoundTripError       bool
		testRoundTripperPreStart bool
		tlsCerts                 *tls.Certificate // for test server
	}{
		{
			name: "successful authentication with valid TLS config",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"proxy_disable":           true,
						"timeout":                 "60s",
						"idle_connection_timeout": "3s",
						"ssl": map[string]any{
							"enabled":           "true",
							"verification_mode": "full",
							"certificate_authorities": []string{
								string(
									pem.EncodeToMemory(&pem.Block{
										Type:  "CERTIFICATE",
										Bytes: caCert.Leaf.Raw,
									})),
							},
						},
					},
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusOK,
			expectHTTPClientType: "httpClientProvider",
			testHTTPRequest:      true,
			tlsCerts:             &serverCerts,
		},
		{
			name: "invalid TLS certificate - continueOnError false",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"ssl": map[string]any{
							"enabled":     "true",
							"certificate": "/nonexistent/cert.pem",
							"key":         "/nonexistent/key.pem",
						},
					},
					ContinueOnError: false,
				}
			},
			expectStartError: true,
			expectStatus:     componentstatus.StatusPermanentError,
		},
		{
			name: "invalid TLS certificate - continueOnError true",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"ssl": map[string]any{
							"enabled":     "true",
							"certificate": "/nonexistent/cert.pem",
							"key":         "/nonexistent/key.pem",
						},
					},
					ContinueOnError: true,
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusPermanentError,
			expectHTTPClientType: "errorRoundTripperProvider",
			testRoundTripError:   true,
		},
		{
			name: "successful client creation with minimal config",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{},
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusOK,
			expectHTTPClientType: "httpClientProvider",
		},
		{
			name: "RoundTripper called before Start",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{},
				}
			},
			skipStart:                true,
			testRoundTripperPreStart: true,
		},
		{
			name: "invalid kerberos auth type - continueOnError true",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"kerberos": map[string]any{
							"auth_type": "invalid_auth_type",
						},
					},
					ContinueOnError: true,
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusPermanentError,
			expectHTTPClientType: "errorRoundTripperProvider",
			testRoundTripError:   true,
		},
		{
			name: "valid kerberos config",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"kerberos": map[string]any{
							"auth_type":   "password",
							"config_path": "../../../../libbeat/outputs/elasticsearch/testdata/krb5.conf",
							"username":    "user",
							"password":    "pass",
							"realm":       "elastic",
						},
					},
					ContinueOnError: true,
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusOK,
			expectHTTPClientType: "kerberosClientProvider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := componenttest.NewNopTelemetrySettings()
			settings.Logger = zaptest.NewLogger(t)
			cfg := tc.setupConfig(t)

			auth, err := newAuthenticator(cfg, settings)
			require.NoError(t, err)

			if tc.testRoundTripperPreStart {
				rt, err := auth.RoundTripper(nil)
				require.Error(t, err)
				require.Nil(t, rt)
				require.Contains(t, err.Error(), "authenticator not started")
				return
			}

			if tc.skipStart {
				return
			}

			var reportedStatuses []componentstatus.Status
			host := &mockHost{
				extensions: extensionsMap{component.NewID(Type): auth},
				reportStatusFunc: func(ev *componentstatus.Event) {
					reportedStatuses = append(reportedStatuses, ev.Status())
				},
			}

			err = auth.Start(context.Background(), host)
			if tc.expectStartError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Validate component status
			require.Len(t, reportedStatuses, 1)
			require.Equal(t, tc.expectStatus, reportedStatuses[0])

			// Validate provider type if specified
			if tc.expectHTTPClientType != "" {
				require.NotNil(t, auth.rtProvider)

				switch tc.expectHTTPClientType {
				case "httpClientProvider":
					_, ok := (auth.rtProvider).(*httpClientProvider)
					require.True(t, ok, "Provider should be an httpClientProvider")
				case "errorRoundTripperProvider":
					_, ok := (auth.rtProvider).(*errorRoundTripperProvider)
					require.True(t, ok, "Provider should be an errorRoundTripperProvider")
				case "kerberosClientProvider":
					_, ok := (auth.rtProvider).(*kerberosClientProvider)
					require.True(t, ok, "Provider should be a kerberosClientProvider")
				}

				rt, err := auth.RoundTripper(nil)
				require.NoError(t, err)
				require.NotNil(t, rt)

				if tc.expectHTTPClientType == "errorRoundTripperProvider" {
					_, ok := rt.(*errorRoundTripper)
					require.True(t, ok, "RoundTripper should be an errorRoundTripper")
				}
			}

			// Test RoundTrip error if specified
			if tc.testRoundTripError {
				rt, err := auth.RoundTripper(nil)
				require.NoError(t, err)

				req, err := http.NewRequest("GET", "http://example.com", nil) //nolint:noctx // this is only in test
				require.NoError(t, err)
				resp, err := rt.RoundTrip(req) //nolint:bodyclose // response is nil
				require.Error(t, err)
				require.Nil(t, resp)
				require.Contains(t, err.Error(), "failed")
			}

			// Test HTTP request if specified
			if tc.testHTTPRequest {
				require.NotNil(t, tc.tlsCerts, "tlsCerts must be provided for testHTTPRequest")

				serverURL := startTestServer(t, []tls.Certificate{*tc.tlsCerts})

				httpClientConfig := confighttp.NewDefaultClientConfig()
				httpClientConfig.Auth = configoptional.Some(configauth.Config{
					AuthenticatorID: component.NewID(Type),
				})

				client, err := httpClientConfig.ToClient(context.Background(), host.GetExtensions(), settings)
				require.NoError(t, err)
				require.NotNil(t, client)

				resp, err := client.Get(serverURL) //nolint:noctx // this is a test
				require.NoError(t, err)
				_ = resp.Body.Close()
			}
		})
	}
}

// startTestServer starts a HTTP server for testing using the provided
// certificates
//
// All requests are responded with an HTTP 200 OK and a plain
// text string
//
// The HTTP server will shutdown at the end of the test.
func startTestServer(t *testing.T, serverCerts []tls.Certificate) string {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("SSL test server")); err != nil {
			t.Errorf("coluld not write to client: %s", err)
		}
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: serverCerts,
	}
	server.StartTLS()
	t.Cleanup(func() { server.Close() })
	return server.URL
}

// TestCertificateHotReload verifies that the beatsauth extension picks up a
// rotated client certificate from disk without restarting the process.
func TestCertificateHotReload(t *testing.T) {
	caCert, err := tlscommontest.GenCA()
	require.NoError(t, err)

	// Two distinct client certs, both signed by the same CA so the server
	// trusts both. Rotation = swap A's PEM files for B's on disk.
	// Using CommonName to assert on the presented cert.
	const (
		clientACN = "beatsauth-client-a"
		clientBCN = "beatsauth-client-b"
	)
	clientCertA, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageDigitalSignature, false, clientACN, []string{}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err)
	clientCertB, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageDigitalSignature, false, clientBCN, []string{}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err)
	require.NotEqual(t, clientCertA.Leaf.Subject.CommonName, clientCertB.Leaf.Subject.CommonName,
		"test setup expects two distinct client certs")

	serverCert, err := tlscommontest.GenSignedCert(caCert, x509.KeyUsageCertSign, false, "", []string{}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Leaf.Raw})

	startMTLSServer := func(t *testing.T) (serverURL string, lastCommonName func() string) {
		t.Helper()
		caPool := x509.NewCertPool()
		caPool.AddCert(caCert.Leaf)

		var mu sync.Mutex
		var commonName string
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			if len(r.TLS.PeerCertificates) > 0 {
				commonName = r.TLS.PeerCertificates[0].Subject.CommonName
			}
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		srv.TLS = &tls.Config{
			MinVersion:             tls.VersionTLS12,
			Certificates:           []tls.Certificate{serverCert},
			ClientAuth:             tls.RequireAndVerifyClientCert,
			ClientCAs:              caPool,
			SessionTicketsDisabled: true,
		}
		srv.StartTLS()
		t.Cleanup(srv.Close)
		return srv.URL, func() string {
			mu.Lock()
			defer mu.Unlock()
			return commonName
		}
	}

	const idleConnTimeout = 10 * time.Millisecond
	const interReqSleep = 50 * time.Millisecond

	startAuthClient := func(t *testing.T, beatAuthConfig map[string]any) (*http.Client, *observer.ObservedLogs) {
		t.Helper()
		core, observed := observer.New(zapcore.WarnLevel)

		// Always inject the short idle timeout so the connection pool is
		// drained between requests in doGET.
		beatAuthConfig["idle_connection_timeout"] = idleConnTimeout.String()

		settings := componenttest.NewNopTelemetrySettings()
		settings.Logger = zap.New(core)
		auth, err := newAuthenticator(&Config{BeatAuthConfig: beatAuthConfig}, settings)
		require.NoError(t, err)

		host := &mockHost{
			extensions:       extensionsMap{component.NewID(Type): auth},
			reportStatusFunc: func(*componentstatus.Event) {},
		}
		require.NoError(t, auth.Start(t.Context(), host))

		httpClientCfg := confighttp.NewDefaultClientConfig()
		httpClientCfg.Auth = configoptional.Some(configauth.Config{
			AuthenticatorID: component.NewID(Type),
		})
		client, err := httpClientCfg.ToClient(t.Context(), host.GetExtensions(), settings)
		require.NoError(t, err)
		return client, observed
	}

	doGET := func(t *testing.T, client *http.Client, url string) {
		t.Helper()
		resp, err := client.Get(url) //nolint:noctx // test
		require.NoError(t, err)
		_ = resp.Body.Close()
		time.Sleep(interReqSleep)
	}

	sslCfg := func(certPath, keyPath string, extras map[string]any) map[string]any {
		ssl := map[string]any{
			"enabled":                 "true",
			"certificate":             certPath,
			"key":                     keyPath,
			"certificate_authorities": []string{string(caPEM)},
		}
		maps.Copy(ssl, extras)
		return map[string]any{"ssl": ssl}
	}

	t.Run("rotates the presented cert when reload is on (default)", func(t *testing.T) {
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		serverURL, lastCN := startMTLSServer(t)

		client, _ := startAuthClient(t, sslCfg(certPath, keyPath, map[string]any{
			"certificate_reload": map[string]any{
				"reload_interval": "100ms",
			},
		}))

		doGET(t, client, serverURL)
		require.Equal(t, clientACN, lastCN(), "server should initially see cert A")

		writeCertKeyFiles(t, certPath, keyPath, clientCertB)

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && lastCN() != clientBCN {
			doGET(t, client, serverURL)
		}
		require.Equal(t, clientBCN, lastCN(),
			"server should see cert B after the reload interval elapses")
	})

	t.Run("does not rotate when reload is explicitly disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		serverURL, lastCN := startMTLSServer(t)

		client, _ := startAuthClient(t, sslCfg(certPath, keyPath, map[string]any{
			"certificate_reload": map[string]any{
				"enabled": false,
			},
		}))

		doGET(t, client, serverURL)
		require.Equal(t, clientACN, lastCN(), "server should initially see cert A")

		writeCertKeyFiles(t, certPath, keyPath, clientCertB)

		// With reload disabled the lib loads the cert once at startup, so
		// after rotation the in-memory cert stays at A. Drive several
		// round-trips well past any plausible reload window.
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			doGET(t, client, serverURL)
			require.Equal(t, clientACN, lastCN(),
				"server should keep seeing cert A when reload is disabled")
		}
	})

	t.Run("legacy restart_on_cert_change.enabled=true aliases reload on", func(t *testing.T) {
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		serverURL, lastCN := startMTLSServer(t)

		// Only the legacy key is set and no ssl.certificate_reload block. The
		// alias must enable reload and seed reload_interval from .period so
		// the rotation finishes within the test deadline.
		client, observed := startAuthClient(t, sslCfg(certPath, keyPath, map[string]any{
			"restart_on_cert_change": map[string]any{
				"enabled": true,
				"period":  "100ms",
			},
		}))

		warnings := observed.FilterMessageSnippet("'ssl.restart_on_cert_change' is deprecated").All()
		require.Len(t, warnings, 1, "expected one deprecation warning")

		doGET(t, client, serverURL)
		require.Equal(t, clientACN, lastCN())

		writeCertKeyFiles(t, certPath, keyPath, clientCertB)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && lastCN() != clientBCN {
			doGET(t, client, serverURL)
		}
		require.Equal(t, clientBCN, lastCN(),
			"alias should enable reload with the legacy period as reload_interval")
	})

	t.Run("ssl.enabled=false with certificate_reload.enabled=true does not rotate certs", func(t *testing.T) {
		// tlscommon guards CertificateReload behind its own IsEnabled() check,
		// so when ssl.enabled=false the hot-reloader is never started regardless
		// of the certificate_reload block. goleak in TestMain enforces that no
		// reloader goroutine leaks out of this subtest.
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		client, _ := startAuthClient(t, map[string]any{
			"ssl": map[string]any{
				"enabled":                 false,
				"certificate":             certPath,
				"key":                     keyPath,
				"certificate_authorities": []string{string(caPEM)},
				"certificate_reload": map[string]any{
					"enabled":         true,
					"reload_interval": "100ms",
				},
			},
		})

		// Rotate the cert on disk — if a reloader were running it would pick
		// this up and the goroutine would outlive the subtest, failing goleak.
		writeCertKeyFiles(t, certPath, keyPath, clientCertB)
		time.Sleep(300 * time.Millisecond) // longer than reload_interval

		// The client must still be usable (no panic, no internal error).
		require.NotNil(t, client)
	})

	t.Run("ssl.enabled=false skips restart_on_cert_change alias entirely", func(t *testing.T) {
		// When ssl.enabled is explicitly false, the TLS config is present but
		// disabled. The alias must not touch CertificateReload in that case.
		//
		// We verify by checking that no deprecation warning is emitted even
		// though restart_on_cert_change keys are present.
		beatAuthCfg := map[string]any{
			"ssl": map[string]any{
				"enabled": false,
				"restart_on_cert_change": map[string]any{
					"enabled": true,
					"period":  "100ms",
				},
			},
		}

		core, observed := observer.New(zapcore.WarnLevel)
		settings := componenttest.NewNopTelemetrySettings()
		settings.Logger = zap.New(core)
		auth, err := newAuthenticator(&Config{BeatAuthConfig: beatAuthCfg}, settings)
		require.NoError(t, err)

		host := &mockHost{
			extensions:       extensionsMap{component.NewID(Type): auth},
			reportStatusFunc: func(*componentstatus.Event) {},
		}
		require.NoError(t, auth.Start(t.Context(), host))

		warnings := observed.FilterMessageSnippet("'ssl.restart_on_cert_change' is deprecated").All()
		require.Empty(t, warnings, "alias must not fire when ssl.enabled=false")
	})

	t.Run("certificate_reload.enabled=false takes precedence over restart_on_cert_change.enabled=true", func(t *testing.T) {
		// When both options are present, certificate_reload wins.
		// The alias code only applies restart_on_cert_change values when
		// certificate_reload has not been explicitly set (Enabled == nil).
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		serverURL, lastCN := startMTLSServer(t)

		client, observed := startAuthClient(t, sslCfg(certPath, keyPath, map[string]any{
			"certificate_reload": map[string]any{
				"enabled": false,
			},
			"restart_on_cert_change": map[string]any{
				"enabled": true,
				"period":  "100ms",
			},
		}))

		warnings := observed.FilterMessageSnippet("'ssl.restart_on_cert_change' is deprecated").All()
		require.Len(t, warnings, 1, "expected one deprecation warning")

		doGET(t, client, serverURL)
		require.Equal(t, clientACN, lastCN(), "server should initially see cert A")

		writeCertKeyFiles(t, certPath, keyPath, clientCertB)

		// certificate_reload.enabled=false wins: no rotation despite restart_on_cert_change saying enabled=true.
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			doGET(t, client, serverURL)
			require.Equal(t, clientACN, lastCN(),
				"certificate_reload=false must prevent rotation even when restart_on_cert_change=true")
		}
	})

	t.Run("legacy restart_on_cert_change.enabled=false aliases reload off", func(t *testing.T) {
		tmpDir := t.TempDir()
		certPath := filepath.Join(tmpDir, "client.crt")
		keyPath := filepath.Join(tmpDir, "client.key")
		writeCertKeyFiles(t, certPath, keyPath, clientCertA)

		serverURL, lastCN := startMTLSServer(t)

		// Users who explicitly disabled the legacy key must still get reload
		// disabled, even though ssl.certificate_reload defaults to enabled
		// upstream.
		client, observed := startAuthClient(t, sslCfg(certPath, keyPath, map[string]any{
			"restart_on_cert_change": map[string]any{
				"enabled": false,
			},
		}))

		warnings := observed.FilterMessageSnippet("'ssl.restart_on_cert_change' is deprecated").All()
		require.Len(t, warnings, 1, "expected one deprecation warning")

		doGET(t, client, serverURL)
		require.Equal(t, clientACN, lastCN())

		writeCertKeyFiles(t, certPath, keyPath, clientCertB)
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			doGET(t, client, serverURL)
			require.Equal(t, clientACN, lastCN(),
				"alias enabled=false should keep the reloader off")
		}
	})
}

// writeCertKeyFiles writes a tls.Certificate's PEM-encoded cert and key to disk.
func writeCertKeyFiles(t *testing.T, certPath, keyPath string, cert tls.Certificate) {
	t.Helper()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Leaf.Raw})
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))

	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))
}

type extensionsMap map[component.ID]component.Component

func (m extensionsMap) GetExtensions() map[component.ID]component.Component {
	return m
}

type mockHost struct {
	extensions       extensionsMap
	reportStatusFunc func(*componentstatus.Event)
}

func (m *mockHost) GetExtensions() map[component.ID]component.Component {
	return m.extensions
}

func (m *mockHost) Report(event *componentstatus.Event) {
	if m.reportStatusFunc != nil {
		m.reportStatusFunc(event)
	}
}
