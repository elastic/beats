// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/goleak"
	"go.yaml.in/yaml/v3"

	"github.com/elastic/beats/v7/pkg/transport/tlscommontest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		errText string
	}{
		{
			name: "valid endpoints and API key",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{
					"https://es.example:9200/path?pretty=true",
					"http://es2.example:9200",
				}
				config.APIKey = configopaque.String(base64.StdEncoding.EncodeToString([]byte("id:key")))
				return config
			},
		},
		{
			name: "valid basic authentication",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				config.Password = "password"
				return config
			},
		},
		{
			name: "valid proxy URL with userinfo",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.ProxyURL = "https://proxy-user:proxy-pass@proxy.example:8443"
				return config
			},
		},
		{
			name: "valid Authorization header without explicit credentials",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Headers = configopaque.MapList{
					{Name: "Authorization", Value: "Bearer token"},
				}
				return config
			},
		},
		{
			name: "valid custom header with API key",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Headers = configopaque.MapList{{Name: "X-Custom-Header", Value: "secret"}}
				config.APIKey = "key"
				return config
			},
		},
		{
			name: "missing endpoints",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = nil
				return config
			},
			errText: "at least one endpoint must be configured",
		},
		{
			name: "empty endpoint",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{""}
				return config
			},
			errText: "URL must not be empty",
		},
		{
			name: "unsupported endpoint scheme",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"tcp://es.example:9200"}
				return config
			},
			errText: "scheme must be http or https",
		},
		{
			name: "endpoint without hostname",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"http://:9200"}
				return config
			},
			errText: "URL host must not be empty",
		},
		{
			name: "endpoint fragment",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://es.example:9200#fragment"}
				return config
			},
			errText: "URL fragments are unsupported",
		},
		{
			name: "endpoint userinfo",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://url-user:url-password@es.example:9200"}
				return config
			},
			errText: "endpoint userinfo is unsupported; configure user and password instead",
		},
		{
			name: "user without password",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				return config
			},
			errText: "user and password must be configured together",
		},
		{
			name: "password without user",
			config: func() *Config {
				config := validConfig()
				config.Password = "password"
				return config
			},
			errText: "user and password must be configured together",
		},
		{
			name: "API key and basic auth conflict",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				config.Password = "password"
				config.APIKey = "key"
				return config
			},
			errText: "api_key cannot be combined with basic authentication",
		},
		{
			name: "Authorization header and API key conflict",
			config: func() *Config {
				config := validConfig()
				config.APIKey = "key"
				config.ClientConfig.Headers = configopaque.MapList{
					{Name: "Authorization", Value: "Bearer token"},
				}
				return config
			},
			errText: "Authorization header cannot be combined",
		},
		{
			name: "lower-case authorization header and basic auth conflict",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				config.Password = "password"
				config.ClientConfig.Headers = configopaque.MapList{
					{Name: "authorization", Value: "Bearer token"},
				}
				return config
			},
			errText: "Authorization header cannot be combined",
		},
		{
			name: "nested auth is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Auth = configoptional.Some(configauth.Config{})
				return config
			},
			errText: "nested auth is unsupported",
		},
		{
			name: "singular endpoint is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Endpoint = "https://es.example:9200"
				return config
			},
			errText: "endpoint is unsupported",
		},
		{
			name: "timeout is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Timeout = time.Second
				return config
			},
			errText: "timeout is unsupported",
		},
		{
			name: "middlewares are rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Middlewares = []configmiddleware.Config{{}}
				return config
			},
			errText: "middlewares are unsupported",
		},
		{
			name: "cookies are rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Cookies = configoptional.Some(confighttp.CookiesConfig{})
				return config
			},
			errText: "cookies are unsupported",
		},
		{
			name: "compression is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Compression = configcompression.TypeGzip
				return config
			},
			errText: "compression is unsupported",
		},
		{
			name: "tls.insecure is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.TLS.Insecure = true
				return config
			},
			errText: "tls.insecure is unsupported",
		},
		{
			name: "multiple Authorization headers are rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Headers = configopaque.MapList{
					{Name: "Authorization", Value: "Bearer token"},
					{Name: "authorization", Value: "Bearer other"},
				}
				return config
			},
			errText: "multiple Authorization headers are unsupported",
		},
		{
			name: "invalid proxy URL",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.ProxyURL = "tcp://proxy.example:8080"
				return config
			},
			errText: "invalid proxy_url",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config().Validate()
			if test.errText == "" {
				assert.NoError(t, err, "valid configuration must validate")
				return
			}
			assert.ErrorContains(t, err, test.errText, "configuration must reject invalid input")
		})
	}
}

func TestConfigYAMLDecode(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		verify func(t *testing.T, config *Config)
	}{
		{
			name: "supported fields decode alongside endpoints and credentials",
			yaml: `
endpoints: [https://es.example:9200]
user: elastic
password: password
proxy_url: https://proxy.example:8443
headers:
  X-Test: value
keepalive:
  idle_conn_timeout: 5s
`,
			verify: func(t *testing.T, config *Config) {
				assert.Equal(t, []string{"https://es.example:9200"}, config.Endpoints, "endpoints must not be swallowed by HTTP config decoding")
				assert.Equal(t, "elastic", config.User, "user must decode with transport settings")
				assert.Equal(t, configopaque.String("password"), config.Password, "password must decode with transport settings")
				assert.Equal(t, "https://proxy.example:8443", config.ClientConfig.ProxyURL, "proxy_url must decode with extension fields")

				header, found := config.ClientConfig.Headers.Get("X-Test")
				assert.True(t, found, "configured header must decode")
				assert.Equal(t, configopaque.String("value"), header, "configured header value must decode")

				transport, err := newAuthenticator(config, nil).newTransport()
				require.NoError(t, err, "transport construction must succeed")
				assert.Equal(t, 5*time.Second, transport.IdleConnTimeout, "keepalive settings must reach the transport")
			},
		},
		{
			name: "API key, custom header and insecure_skip_verify",
			yaml: `
endpoints: [https://es.example:9200, https://es2.example:9200]
api_key: some-api-key
headers:
  X-Custom-Header: custom-value
tls:
  insecure_skip_verify: true
`,
			verify: func(t *testing.T, config *Config) {
				assert.Equal(t, []string{"https://es.example:9200", "https://es2.example:9200"}, config.Endpoints, "all endpoints must decode")
				assert.Equal(t, configopaque.String("some-api-key"), config.APIKey, "api_key must decode")

				header, found := config.ClientConfig.Headers.Get("X-Custom-Header")
				assert.True(t, found, "X-Custom-Header header must decode")
				assert.Equal(t, configopaque.String("custom-value"), header, "X-Custom-Header value must decode")

				assert.True(t, config.ClientConfig.TLS.InsecureSkipVerify, "tls.insecure_skip_verify must decode")
			},
		},
		{
			name: "default keepalive settings reach the transport",
			yaml: `
endpoints: [https://es.example:9200]
`,
			verify: func(t *testing.T, config *Config) {
				transport, err := newAuthenticator(config, nil).newTransport()
				require.NoError(t, err, "transport construction must succeed")
				assert.False(t, transport.DisableKeepAlives, "keep-alives must be enabled by default")
				assert.Equal(t, 100, transport.MaxIdleConns, "default MaxIdleConns must reach the transport")
				assert.Equal(t, 90*time.Second, transport.IdleConnTimeout, "default IdleConnTimeout must reach the transport")
			},
		},
		{
			name: "keepalive enabled false disables keep-alives",
			yaml: `
endpoints: [https://es.example:9200]
keepalive:
  enabled: false
`,
			verify: func(t *testing.T, config *Config) {
				transport, err := newAuthenticator(config, nil).newTransport()
				require.NoError(t, err, "transport construction must succeed")
				assert.True(t, transport.DisableKeepAlives, "keepalive.enabled=false must disable keep-alives")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := decodeConfig(t, test.yaml)
			test.verify(t, config)
			assert.NoError(t, config.Validate(), "supported decoded configuration must validate")
		})
	}
}

func TestProgrammaticKeepaliveNoneDisablesKeepAlives(t *testing.T) {
	config := validConfig()
	config.ClientConfig.Keepalive = configoptional.None[confighttp.KeepaliveClientConfig]()

	transport, err := newAuthenticator(config, nil).newTransport()
	require.NoError(t, err, "transport construction must succeed")
	assert.True(t, transport.DisableKeepAlives, "unset Keepalive must disable keep-alives")
}

func TestAuthenticationRoundTrip(t *testing.T) {
	const apiKey = "id:key-base64" //nolint:gosec // test fixture, not a credential.
	tests := []struct {
		name         string
		configure    func(*Config)
		expectAuth   string
		expectHost   string
		expectHeader map[string]string
	}{
		{
			name: "API key",
			configure: func(config *Config) {
				config.APIKey = apiKey
			},
			expectAuth: "ApiKey " + apiKey,
		},
		{
			name: "basic authentication",
			configure: func(config *Config) {
				config.User = "elastic"
				config.Password = "password"
			},
			expectAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("elastic:password")),
		},
		{
			name: "custom header",
			configure: func(config *Config) {
				config.APIKey = apiKey
				config.ClientConfig.Headers = append(config.ClientConfig.Headers,
					configopaque.Pair{Name: "X-Custom-Header", Value: "custom-value"})
			},
			expectAuth:   "ApiKey " + apiKey,
			expectHeader: map[string]string{"X-Custom-Header": "custom-value"},
		},
		{
			name: "case-insensitive host header overrides request host",
			configure: func(config *Config) {
				config.ClientConfig.Headers = append(config.ClientConfig.Headers,
					configopaque.Pair{Name: "host", Value: "es.override.example"})
			},
			expectHost: "es.override.example",
		},
		{
			name: "unauthenticated",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type receivedRequest struct {
				host   string
				header http.Header
			}
			received := make(chan receivedRequest, 1)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received <- receivedRequest{host: r.Host, header: r.Header.Clone()}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			config := validConfig()
			config.Endpoints = []string{server.URL}
			config.ClientConfig.Headers = configopaque.MapList{{Name: "X-Extension", Value: "extension"}}
			if test.configure != nil {
				test.configure(config)
			}

			roundTripper := newTestRoundTripper(t, config)

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err, "request creation must succeed")
			request.Header.Set("X-Caller", "caller")
			originalHost := request.Host

			response, err := roundTripper.RoundTrip(request)
			require.NoError(t, err, "authenticated request must succeed")
			require.NoError(t, response.Body.Close(), "response body must close")

			actual := <-received
			assert.Equal(t, test.expectAuth, actual.header.Get("Authorization"), "transport must inject configured authentication")
			assert.Equal(t, "extension", actual.header.Get("X-Extension"), "transport must inject configured headers")
			assert.Equal(t, "caller", actual.header.Get("X-Caller"), "caller headers must be preserved")
			for name, value := range test.expectHeader {
				assert.Equal(t, value, actual.header.Get(name), "transport must inject header %s", name)
			}
			if test.expectHost != "" {
				assert.Equal(t, test.expectHost, actual.host, "host header must override the request host")
			} else {
				assert.Equal(t, originalHost, actual.host, "request host must remain unchanged without a host header")
			}

			assert.Empty(t, request.Header.Get("Authorization"), "caller request must not gain authentication")
			assert.Empty(t, request.Header.Get("X-Extension"), "caller request must not gain extension headers")
			assert.Equal(t, originalHost, request.Host, "caller request host must remain unchanged")
		})
	}
}

func TestRoundTripNilRequestHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Seen-Authorization", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := validConfig()
	config.Endpoints = []string{server.URL}
	config.APIKey = "key"
	roundTripper := newTestRoundTripper(t, config)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err, "request creation must succeed")
	request.Header = nil

	response, err := roundTripper.RoundTrip(request)
	require.NoError(t, err, "request with nil header must succeed")
	require.NoError(t, response.Body.Close(), "response body must close")
	assert.Equal(t, "ApiKey key", response.Header.Get("X-Seen-Authorization"), "authentication must be injected into a nil header")
}

func TestTLSConfiguration(t *testing.T) {
	ca, err := tlscommontest.GenCA()
	require.NoError(t, err, "CA generation must succeed")
	caFile := writePEMCertificate(t, "ca.pem", ca.Certificate[0])

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	t.Run("custom CA is trusted", func(t *testing.T) {
		server := newTLSServer(t, ca, false, handler)
		config := validConfig()
		config.Endpoints = []string{server.URL}
		config.ClientConfig.TLS.CAFile = caFile

		response, err := doGet(t, newTestRoundTripper(t, config), server.URL)
		require.NoError(t, err, "configured custom CA must trust the HTTPS server")
		assert.Equal(t, http.StatusTeapot, response.StatusCode, "HTTPS request must reach the test handler")
		require.NoError(t, response.Body.Close(), "response body must close")
	})

	t.Run("unknown CA is rejected", func(t *testing.T) {
		server := newTLSServer(t, ca, false, handler)
		config := validConfig()
		config.Endpoints = []string{server.URL}

		response, err := doGet(t, newTestRoundTripper(t, config), server.URL)
		if err == nil {
			_ = response.Body.Close()
		}
		assert.ErrorContains(t, err, "certificate", "server certificate must not be trusted without the CA")
	})

	t.Run("mTLS client certificate", func(t *testing.T) {
		server := newTLSServer(t, ca, true, handler)
		clientCert, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "client", nil, nil, false)
		require.NoError(t, err, "client certificate generation must succeed")
		certFile, keyFile := writeCertificateAndKey(t, clientCert)

		config := validConfig()
		config.Endpoints = []string{server.URL}
		config.ClientConfig.TLS.CAFile = caFile
		config.ClientConfig.TLS.CertFile = certFile
		config.ClientConfig.TLS.KeyFile = keyFile

		response, err := doGet(t, newTestRoundTripper(t, config), server.URL)
		require.NoError(t, err, "client certificate must satisfy the server")
		assert.Equal(t, http.StatusTeapot, response.StatusCode, "mTLS request must reach the test handler")
		require.NoError(t, response.Body.Close(), "response body must close")

		config = validConfig()
		config.Endpoints = []string{server.URL}
		config.ClientConfig.TLS.CAFile = caFile
		response, err = doGet(t, newTestRoundTripper(t, config), server.URL)
		if err == nil {
			_ = response.Body.Close()
		}
		assert.Error(t, err, "server must reject a client without a certificate")
	})

	t.Run("insecure_skip_verify", func(t *testing.T) {
		server := newTLSServer(t, ca, false, handler)
		config := decodeConfig(t, fmt.Sprintf(`
endpoints: [%q]
tls:
  insecure_skip_verify: true
`, server.URL))

		response, err := doGet(t, newTestRoundTripper(t, config), server.URL)
		require.NoError(t, err, "insecure_skip_verify must accept an untrusted certificate")
		assert.Equal(t, http.StatusTeapot, response.StatusCode, "request must reach the test handler")
		require.NoError(t, response.Body.Close(), "response body must close")
	})

	failures := []struct {
		name      string
		configure func(t *testing.T, config *Config)
	}{
		{
			name: "missing CA file",
			configure: func(t *testing.T, config *Config) {
				config.ClientConfig.TLS.CAFile = filepath.Join(t.TempDir(), "missing-ca.pem")
			},
		},
		{
			name: "malformed CA file",
			configure: func(t *testing.T, config *Config) {
				path := filepath.Join(t.TempDir(), "ca.pem")
				require.NoError(t, os.WriteFile(path, []byte("not a certificate"), 0o600), "malformed CA fixture must write")
				config.ClientConfig.TLS.CAFile = path
			},
		},
		{
			name: "mismatched client certificate and key",
			configure: func(t *testing.T, config *Config) {
				certA, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "a", nil, nil, false)
				require.NoError(t, err, "certificate generation must succeed")
				certB, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "b", nil, nil, false)
				require.NoError(t, err, "certificate generation must succeed")
				config.ClientConfig.TLS.CertFile, _ = writeCertificateAndKey(t, certA)
				_, config.ClientConfig.TLS.KeyFile = writeCertificateAndKey(t, certB)
			},
		},
	}
	for _, test := range failures {
		t.Run(test.name+" fails creation only", func(t *testing.T) {
			config := validConfig()
			test.configure(t, config)
			assert.NoError(t, config.Validate(), "Validate must remain independent of TLS files")

			ext, err := createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
			assert.Nil(t, ext, "extension must not be created with invalid TLS material")
			assert.ErrorContains(t, err, "invalid Elasticsearch TLS configuration", "creation must preflight TLS files")
		})
	}
}

func TestProxyURLRouting(t *testing.T) {
	receivedURL := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL <- r.URL.String()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()

	config := validConfig()
	config.ClientConfig.ProxyURL = proxy.URL
	roundTripper := newTestRoundTripper(t, config)

	response, err := doGet(t, roundTripper, "http://unreachable.invalid/_cluster/health")
	require.NoError(t, err, "request must route through the configured proxy")
	require.NoError(t, response.Body.Close(), "response body must close")
	assert.Equal(t, "http://unreachable.invalid/_cluster/health", <-receivedURL, "proxy must receive the absolute request URL")
}

func TestRoundTripperConstructionOwnership(t *testing.T) {
	config := validConfig()
	config.Endpoints = []string{"http://127.0.0.1:1"}
	auth := createTestExtension(t, config)

	first, err := auth.RoundTripper(failingRoundTripper{})
	require.NoError(t, err, "transport construction must not dial Elasticsearch")
	second, err := auth.RoundTripper(failingRoundTripper{})
	require.NoError(t, err, "each transport construction must remain offline")

	firstAuth, ok := first.(*authenticatedRoundTripper)
	require.True(t, ok, "extension must return its request-cloning wrapper")
	secondAuth, ok := second.(*authenticatedRoundTripper)
	require.True(t, ok, "extension must return its request-cloning wrapper")
	assert.NotSame(t, firstAuth.transport, secondAuth.transport, "each consumer must own an independent transport")
	assert.IsType(t, &http.Transport{}, firstAuth.transport, "base transport must be ignored in favor of an owned *http.Transport")

	endpoints := auth.Endpoints()
	assert.Equal(t, config.Endpoints, endpoints, "endpoints must remain configured")
	endpoints[0] = "mutated"
	assert.Equal(t, []string{"http://127.0.0.1:1"}, auth.Endpoints(), "callers must own the returned endpoints slice")
}

func TestRoundTripperCloseIdleConnections(t *testing.T) {
	transport := &closeTrackingRoundTripper{}
	roundTripper := &authenticatedRoundTripper{transport: transport, config: validConfig()}
	roundTripper.CloseIdleConnections()
	assert.Equal(t, 1, transport.closeCalls, "wrapper must forward consumer pool cleanup")
}

func TestCredentialRedaction(t *testing.T) {
	config := validConfig()
	config.User = "elastic"
	config.Password = "do-not-log-password"
	config.APIKey = "do-not-log-api-key"
	config.ClientConfig.Headers = configopaque.MapList{{Name: "X-Custom-Secret", Value: "do-not-log-custom-secret"}}

	formatted := fmt.Sprintf("%+v", config)
	assert.NotContains(t, formatted, "do-not-log-password", "formatted configuration must redact password")
	assert.NotContains(t, formatted, "do-not-log-api-key", "formatted configuration must redact API key")
	assert.NotContains(t, formatted, "do-not-log-custom-secret", "formatted configuration must redact headers")
}

func TestComponentLifecycle(t *testing.T) {
	oteltest.CheckExtensionLifecycle(t, NewFactory(), validConfig())
}

func validConfig() *Config {
	config, ok := createDefaultConfig().(*Config)
	if !ok {
		panic("elasticsearchauth default config has unexpected type")
	}
	config.Endpoints = []string{"https://es.example:9200"}
	return config
}

// decodeConfig decodes a YAML fixture the same way the Collector does, on top
// of the default configuration.
func decodeConfig(t *testing.T, fixture string) *Config {
	t.Helper()
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(fixture), &raw), "YAML fixture must decode")

	config, ok := createDefaultConfig().(*Config)
	require.True(t, ok, "default configuration must be Config")
	require.NoError(t, confmap.NewFromStringMap(raw).Unmarshal(config), "Collector confmap must decode extension configuration")
	return config
}

func createTestExtension(t *testing.T, config *Config) *authenticator {
	t.Helper()
	ext, err := createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
	require.NoError(t, err, "extension creation must succeed")
	auth, ok := ext.(*authenticator)
	require.True(t, ok, "created extension must be authenticator")
	return auth
}

// newTestRoundTripper builds a RoundTripper and closes its idle connections
// when the test ends, so connection goroutines do not leak.
func newTestRoundTripper(t *testing.T, config *Config) http.RoundTripper {
	t.Helper()
	roundTripper, err := createTestExtension(t, config).RoundTripper(failingRoundTripper{})
	require.NoError(t, err, "RoundTripper construction must succeed")
	closer, ok := roundTripper.(interface{ CloseIdleConnections() })
	require.True(t, ok, "RoundTripper must support CloseIdleConnections")
	t.Cleanup(closer.CloseIdleConnections)
	return roundTripper
}

func doGet(t *testing.T, roundTripper http.RoundTripper, url string) (*http.Response, error) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err, "request creation must succeed")
	return roundTripper.RoundTrip(request)
}

// newTLSServer starts an HTTPS server for 127.0.0.1 with a certificate signed
// by ca. With requireClientCert it enforces mTLS against the same CA.
func newTLSServer(t *testing.T, ca tls.Certificate, requireClientCert bool, handler http.Handler) *httptest.Server {
	t.Helper()
	serverCert, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err, "server certificate generation must succeed")

	server := httptest.NewUnstartedServer(handler)
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	if requireClientCert {
		pool := x509.NewCertPool()
		pool.AddCert(ca.Leaf)
		server.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		server.TLS.ClientCAs = pool
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func writePEMCertificate(t *testing.T, name string, rawCertificate []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	contents := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rawCertificate})
	require.NoError(t, os.WriteFile(path, contents, 0o600), "certificate fixture must write")
	return path
}

func writeCertificateAndKey(t *testing.T, certificate tls.Certificate) (string, string) {
	t.Helper()
	directory := t.TempDir()
	certPath := filepath.Join(directory, "cert.pem")
	keyPath := filepath.Join(directory, "key.pem")
	key, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	require.NoError(t, err, "private key marshal must succeed")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}), 0o600), "certificate fixture must write")
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0o600), "private key fixture must write")
	return certPath, keyPath
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("base transport must not be used")
}

type closeTrackingRoundTripper struct {
	closeCalls int
}

func (*closeTrackingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("round trip is not used by close test")
}

func (c *closeTrackingRoundTripper) CloseIdleConnections() {
	c.closeCalls++
}
