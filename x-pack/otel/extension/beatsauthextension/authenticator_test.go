// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.uber.org/goleak"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, []goleak.Option{
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreTopFunction("net/http.(*persistConn).writeLoop")}...)

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
						"loadbalance":             true,
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
					BeatAuthConfig: map[string]any{
						"loadbalance": true,
					},
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
		{
			name: "when loadbalance is false and endpoints are not configured",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"loadbalance": false,
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
			name: "when loadbalance is false and endpoints are configured",
			setupConfig: func(t *testing.T) *Config {
				return &Config{
					BeatAuthConfig: map[string]any{
						"loadbalance": false,
						"hosts": []string{
							"http://localhost:9200",
						},
					},
					ContinueOnError: true,
				}
			},
			expectStartError:     false,
			expectStatus:         componentstatus.StatusOK,
			expectHTTPClientType: "singleRouterProvider",
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
				case "singleRouterProvider":
					_, ok := (auth.rtProvider).(*singleRouterProvider)
					require.True(t, ok, "Provider should be a singleRouterProvider")
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

				client, err := httpClientConfig.ToClient(context.Background(), host, settings)
				require.NoError(t, err)
				require.NotNil(t, client)

				resp, err := client.Get(serverURL) //nolint:noctx // this is a test
				require.NoError(t, err)
				_ = resp.Body.Close()
			}
		})
	}
}

func TestSingleRouterProvider(t *testing.T) {

	var requestReceived bool
	var startServer = func(url string, requestReceived *bool) {
		l, err := net.Listen("tcp", url)
		require.NoError(t, err)
		server := &http.Server{ //nolint:gosec // testing
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				*requestReceived = true
				if _, err := w.Write([]byte("Hello, World!")); err != nil {
					t.Errorf("could not write to client: %s", err)
				}
			}),
		}

		// Start server and shut it down when the tests are over.
		go func() {
			_ = server.Serve(l)
		}()

		t.Cleanup(
			func() {
				if l == nil {
					return
				} else {
					_ = server.Close()
				}
			},
		)
	}

	cfg := &Config{
		BeatAuthConfig: map[string]any{
			"loadbalance": false,
			"hosts": []string{
				"http://localhost:8080",
				"http://localhost:8090",
			},
		},
	}
	settings := componenttest.NewNopTelemetrySettings()
	auth, err := newAuthenticator(cfg, settings)
	require.NoError(t, err)

	err = auth.Start(context.Background(), nil)
	require.NoError(t, err)

	startServer("localhost:8080", &requestReceived)

	rt, err := auth.RoundTripper(nil)
	require.NoError(t, err)

	// we set wrong endpoint to see if it uses the correct one set on beatsauth
	req1, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = rt.RoundTrip(req1)
	// if err is not nil, we retry again and this time it should connect to the active server
	if err != nil {
		_, err = rt.RoundTrip(req1)
		require.NoError(t, err)
	}
	require.Equal(t, true, requestReceived)

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
