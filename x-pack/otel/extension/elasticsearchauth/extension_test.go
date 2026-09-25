// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"gopkg.in/yaml.v3"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
)

func TestConfigValidate(t *testing.T) {
	validAPIKey := configopaque.String(base64.StdEncoding.EncodeToString([]byte("id:key")))
	tests := []struct {
		name    string
		config  func() *Config
		errText string
	}{
		{
			name: "valid endpoints and API key",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://es.example:9200/path?pretty=true"}
				config.APIKey = validAPIKey
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
			name: "missing endpoints",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = nil
				return config
			},
			errText: "at least one endpoint",
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
			name: "endpoint fragment",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://es.example:9200#fragment"}
				return config
			},
			errText: "fragments",
		},
		{
			name: "invalid API key base64",
			config: func() *Config {
				config := validConfig()
				config.APIKey = "not-base64"
				return config
			},
			errText: "base64-encoded id:key",
		},
		{
			name: "API key without id and key",
			config: func() *Config {
				config := validConfig()
				config.APIKey = configopaque.String(base64.StdEncoding.EncodeToString([]byte("id")))
				return config
			},
			errText: "base64-encoded id:key",
		},
		{
			name: "user without password",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				return config
			},
			errText: "configured together",
		},
		{
			name: "password without user",
			config: func() *Config {
				config := validConfig()
				config.Password = "password"
				return config
			},
			errText: "configured together",
		},
		{
			name: "API key and basic auth conflict",
			config: func() *Config {
				config := validConfig()
				config.User = "elastic"
				config.Password = "password"
				config.APIKey = validAPIKey
				return config
			},
			errText: "cannot be combined",
		},
		{
			name: "endpoint userinfo and explicit credentials conflict",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://url-user:url-password@es.example:9200"}
				config.User = "elastic"
				config.Password = "password"
				return config
			},
			errText: "userinfo cannot be combined",
		},
		{
			name: "nested auth is rejected",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.Auth = configoptional.Some(configauth.Config{})
				return config
			},
			errText: "nested auth",
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
			name: "invalid proxy URL",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.ProxyURL = "tcp://proxy.example:8080"
				return config
			},
			errText: "invalid proxy_url",
		},
		{
			name: "TLS certificate without key",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.TLS.CertFile = "client.pem"
				return config
			},
			errText: "both certificate and key",
		},
		{
			name: "TLS CA file and PEM conflict",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.TLS.CAFile = "ca.pem"
				config.ClientConfig.TLS.CAPem = "ca pem"
				return config
			},
			errText: "either a CA file or the PEM",
		},
		{
			name: "invalid TLS minimum version",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.TLS.MinVersion = "42"
				return config
			},
			errText: "invalid TLS min_version",
		},
		{
			name: "TLS maximum version below minimum",
			config: func() *Config {
				config := validConfig()
				config.ClientConfig.TLS.MinVersion = "1.3"
				config.ClientConfig.TLS.MaxVersion = "1.2"
				return config
			},
			errText: "min_version cannot be greater",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config().Validate()
			if test.errText == "" {
				require.NoError(t, err, "valid configuration must validate")
				return
			}
			require.ErrorContains(t, err, test.errText, "configuration must reject invalid structural input")
		})
	}
}

func TestConfigYAMLDecode(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		decodeError     string
		expectEndpoints []string
		expectUser      string
		expectPassword  configopaque.String
		expectProxyURL  string
		expectHeader    configopaque.String
		expectIdle      time.Duration
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
			expectEndpoints: []string{"https://es.example:9200"},
			expectUser:      "elastic",
			expectPassword:  "password",
			expectProxyURL:  "https://proxy.example:8443",
			expectHeader:    "value",
			expectIdle:      5 * time.Second,
		},
		{
			name: "singular endpoint",
			yaml: `
endpoints: [https://es.example:9200]
endpoint: https://other.example:9200
`,
			decodeError: "endpoint is unsupported",
		},
		{
			name: "null endpoint",
			yaml: `
endpoints: [https://es.example:9200]
endpoint: null
`,
			decodeError: "endpoint is unsupported",
		},
		{
			name: "timeout",
			yaml: `
endpoints: [https://es.example:9200]
timeout: 1s
`,
			decodeError: "timeout is unsupported",
		},
		{
			name: "zero timeout",
			yaml: `
endpoints: [https://es.example:9200]
timeout: 0s
`,
			decodeError: "timeout is unsupported",
		},
		{
			name: "nested auth",
			yaml: `
endpoints: [https://es.example:9200]
auth:
  authenticator: basicauth/default
`,
			decodeError: "nested auth",
		},
		{
			name: "null auth",
			yaml: `
endpoints: [https://es.example:9200]
auth: null
`,
			decodeError: "nested auth",
		},
		{
			name: "middleware",
			yaml: `
endpoints: [https://es.example:9200]
middlewares:
  - id: basicauth/default
`,
			decodeError: "middlewares are unsupported",
		},
		{
			name: "empty middlewares",
			yaml: `
endpoints: [https://es.example:9200]
middlewares: []
`,
			decodeError: "middlewares are unsupported",
		},
		{
			name: "cookies",
			yaml: `
endpoints: [https://es.example:9200]
cookies: {}
`,
			decodeError: "cookies are unsupported",
		},
		{
			name: "null cookies",
			yaml: `
endpoints: [https://es.example:9200]
cookies: null
`,
			decodeError: "cookies are unsupported",
		},
		{
			name: "compression",
			yaml: `
endpoints: [https://es.example:9200]
compression: gzip
`,
			decodeError: "compression is unsupported",
		},
		{
			name: "empty compression",
			yaml: `
endpoints: [https://es.example:9200]
compression: ''
`,
			decodeError: "compression is unsupported",
		},
		{
			name: "empty compression parameters",
			yaml: `
endpoints: [https://es.example:9200]
compression_params: {}
`,
			decodeError: "compression_params are unsupported",
		},
		{
			name: "unknown key",
			yaml: `
endpoints: [https://es.example:9200]
unknown_key: value
`,
			decodeError: "unsupported configuration key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var raw map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(test.yaml), &raw), "YAML fixture must decode")

			defaultConfig := createDefaultConfig()
			config, ok := defaultConfig.(*Config)
			require.True(t, ok, "default configuration must be Config")

			err := confmap.NewFromStringMap(raw).Unmarshal(config)
			if test.decodeError != "" {
				require.ErrorContains(t, err, test.decodeError, "unsupported or unknown configuration must fail decoding")
				return
			}
			require.NoError(t, err, "Collector confmap must decode extension configuration")

			expectedEndpoints := test.expectEndpoints
			if expectedEndpoints == nil {
				expectedEndpoints = []string{"https://es.example:9200"}
			}
			require.Equal(t, expectedEndpoints, config.Endpoints, "endpoints must not be swallowed by HTTP config decoding")
			require.Equal(t, test.expectUser, config.User, "user must decode with transport settings")
			require.Equal(t, test.expectPassword, config.Password, "password must decode with transport settings")
			require.Equal(t, test.expectProxyURL, config.ClientConfig.ProxyURL, "proxy_url must decode with extension fields")

			header, found := config.ClientConfig.Headers.Get("X-Test")
			require.True(t, found, "configured header must decode")
			require.Equal(t, test.expectHeader, header, "configured header value must decode")

			if test.expectIdle != 0 {
				//nolint:staticcheck // confighttp folds decoded keepalive values into this field.
				require.Equal(t, test.expectIdle, config.ClientConfig.IdleConnTimeout, "keepalive settings must decode")
			}

			require.NoError(t, config.Validate(), "supported decoded configuration must validate")
		})
	}
}

func TestProgrammaticKeepaliveOverridesFlattenedFields(t *testing.T) {
	config := validConfig()
	//nolint:staticcheck // confighttp.ClientConfig documents these as its effective decoded values.
	config.ClientConfig.DisableKeepAlives = true
	//nolint:staticcheck // confighttp.ClientConfig documents these as its effective decoded values.
	config.ClientConfig.MaxIdleConns = 1
	//nolint:staticcheck // confighttp.ClientConfig documents these as its effective decoded values.
	config.ClientConfig.MaxIdleConnsPerHost = 2
	//nolint:staticcheck // confighttp.ClientConfig documents these as its effective decoded values.
	config.ClientConfig.IdleConnTimeout = time.Second
	config.ClientConfig.Keepalive = configoptional.Some(confighttp.KeepaliveClientConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	})

	transport, err := newAuthenticator(config, nil).newTransport()
	require.NoError(t, err)
	require.False(t, transport.DisableKeepAlives)
	require.Equal(t, 10, transport.MaxIdleConns)
	require.Equal(t, 20, transport.MaxIdleConnsPerHost)
	require.Equal(t, 30*time.Second, transport.IdleConnTimeout)
}

func TestAuthenticationRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*Config)
		expectAuth string
	}{
		{
			name: "API key",
			configure: func(config *Config) {
				config.APIKey = configopaque.String(base64.StdEncoding.EncodeToString([]byte("id:key")))
			},
			expectAuth: "ApiKey " + base64.StdEncoding.EncodeToString([]byte("id:key")),
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
			name: "unauthenticated",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type receivedAuth struct {
				authorization string
				extension     string
			}
			received := make(chan receivedAuth, 1)

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				received <- receivedAuth{
					authorization: request.Header.Get("Authorization"),
					extension:     request.Header.Get("X-Extension"),
				}

				response.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			config := validConfig()
			config.Endpoints = []string{server.URL}
			config.ClientConfig.Headers = configopaque.MapList{{Name: "X-Extension", Value: "extension"}}
			if test.configure != nil {
				test.configure(config)
			}

			authenticator := createTestExtension(t, config)
			roundTripper, err := authenticator.RoundTripper(failingRoundTripper{})
			require.NoError(t, err, "RoundTripper construction must succeed")

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err, "request creation must succeed")

			request.Header.Set("X-Caller", "caller")
			response, err := roundTripper.RoundTrip(request)
			require.NoError(t, err, "authenticated request must succeed")
			require.NoError(t, response.Body.Close(), "response body must close")

			actual := <-received
			require.Equal(t, test.expectAuth, actual.authorization, "transport must inject configured authentication")
			require.Equal(t, "extension", actual.extension, "transport must inject configured headers")

			require.Empty(t, request.Header.Get("Authorization"), "caller request must not gain authentication")
			require.Empty(t, request.Header.Get("X-Extension"), "caller request must not gain extension headers")
			require.Equal(t, "caller", request.Header.Get("X-Caller"), "caller request headers must remain unchanged")
		})
	}
}

func TestCreateExtensionTLSPreflight(t *testing.T) {
	tests := []struct {
		name          string
		configure     func(t *testing.T, config *Config)
		validateError string
		createError   string
	}{
		{
			name: "valid CA and client certificate material",
			configure: func(t *testing.T, config *Config) {
				ca, cert := generateCAAndCertificate(t)
				config.ClientConfig.TLS.CAFile = writePEMCertificate(t, "ca.pem", ca.Certificate[0])
				config.ClientConfig.TLS.CertFile, config.ClientConfig.TLS.KeyFile = writeCertificateAndKey(t, cert)
			},
		},
		{
			name: "missing CA file is only rejected during creation",
			configure: func(t *testing.T, config *Config) {
				config.ClientConfig.TLS.CAFile = filepath.Join(t.TempDir(), "missing-ca.pem")
			},
			createError: "invalid Elasticsearch TLS configuration",
		},
		{
			name: "malformed CA file",
			configure: func(t *testing.T, config *Config) {
				path := filepath.Join(t.TempDir(), "ca.pem")
				require.NoError(t, os.WriteFile(path, []byte("not a certificate"), 0o600), "malformed CA fixture must write")
				config.ClientConfig.TLS.CAFile = path
			},
			createError: "invalid Elasticsearch TLS configuration",
		},
		{
			name: "mismatched client certificate and key",
			configure: func(t *testing.T, config *Config) {
				_, certA := generateCAAndCertificate(t)
				_, certB := generateCAAndCertificate(t)
				config.ClientConfig.TLS.CertFile, _ = writeCertificateAndKey(t, certA)
				_, config.ClientConfig.TLS.KeyFile = writeCertificateAndKey(t, certB)
			},
			createError: "invalid Elasticsearch TLS configuration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validConfig()
			test.configure(t, config)
			err := config.Validate()
			if test.validateError == "" {
				require.NoError(t, err, "Validate must remain independent of TLS files")
			} else {
				require.ErrorContains(t, err, test.validateError, "Validate must report structural configuration errors")
			}

			_, err = createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
			if test.createError == "" {
				require.NoError(t, err, "valid TLS material must pass creation preflight")
			} else {
				require.ErrorContains(t, err, test.createError, "creation must preflight TLS files")
			}
		})
	}
}

func TestCustomCATLSRequest(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	certificate := server.Certificate()
	config := validConfig()
	config.Endpoints = []string{server.URL}
	config.ClientConfig.TLS.CAFile = writePEMCertificate(t, "server-ca.pem", certificate.Raw)

	authenticator := createTestExtension(t, config)

	roundTripper, err := authenticator.RoundTripper(nil)
	require.NoError(t, err, "RoundTripper construction must succeed")

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err, "request creation must succeed")

	response, err := roundTripper.RoundTrip(request)
	require.NoError(t, err, "configured custom CA must trust HTTPS server")
	require.Equal(t, http.StatusTeapot, response.StatusCode, "HTTPS request must reach the test handler")
	require.NoError(t, response.Body.Close(), "response body must close")
}

func TestProxyURLRouting(t *testing.T) {
	receivedURL := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		receivedURL <- request.URL.String()
		response.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()

	config := validConfig()
	config.ClientConfig.ProxyURL = proxy.URL
	authenticator := createTestExtension(t, config)
	roundTripper, err := authenticator.RoundTripper(nil)
	require.NoError(t, err, "proxy transport construction must succeed")
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://unreachable.invalid/_cluster/health", nil)
	require.NoError(t, err, "request creation must succeed")
	response, err := roundTripper.RoundTrip(request)
	require.NoError(t, err, "request must route through configured proxy")
	require.NoError(t, response.Body.Close(), "response body must close")
	require.Equal(t, "http://unreachable.invalid/_cluster/health", <-receivedURL, "proxy must receive absolute request URL")
}

func TestRoundTripperConstructionOwnership(t *testing.T) {
	config := validConfig()
	config.Endpoints = []string{"http://127.0.0.1:1"}
	authenticator := createTestExtension(t, config)
	first, err := authenticator.RoundTripper(failingRoundTripper{})
	require.NoError(t, err, "transport construction must not dial Elasticsearch")
	second, err := authenticator.RoundTripper(failingRoundTripper{})
	require.NoError(t, err, "each transport construction must remain offline")
	firstAuth, ok := first.(*authenticatedRoundTripper)
	require.True(t, ok, "extension must return its request-cloning wrapper")
	secondAuth, ok := second.(*authenticatedRoundTripper)
	require.True(t, ok, "extension must return its request-cloning wrapper")
	require.NotSame(t, firstAuth.transport, secondAuth.transport, "each consumer must own an independent transport")
	require.Equal(t, config.Endpoints, authenticator.Endpoints(), "endpoints must remain configured")
}

func TestRoundTripperCloseIdleConnections(t *testing.T) {
	transport := &closeTrackingRoundTripper{}
	roundTripper := &authenticatedRoundTripper{transport: transport, config: validConfig()}
	roundTripper.CloseIdleConnections()
	require.Equal(t, 1, transport.closeCalls, "wrapper must forward consumer pool cleanup")
}

func TestCredentialRedaction(t *testing.T) {
	tests := []struct {
		name            string
		configure       func(*Config)
		secrets         []string
		validationError bool
	}{
		{
			name: "malformed API key validation error",
			configure: func(config *Config) {
				config.APIKey = "do-not-log-malformed-api-key"
			},
			secrets:         []string{"do-not-log-malformed-api-key"},
			validationError: true,
		},
		{
			name: "basic authentication construction error",
			configure: func(config *Config) {
				config.User = "elastic"
				config.Password = "do-not-log-password"
				config.ClientConfig.TLS.CAFile = filepath.Join(t.TempDir(), "missing-ca.pem")
			},
			secrets: []string{"do-not-log-password"},
		},
		{
			name: "API key construction error",
			configure: func(config *Config) {
				config.APIKey = configopaque.String(base64.StdEncoding.EncodeToString([]byte("id:do-not-log-api-key")))
				config.ClientConfig.TLS.CAFile = filepath.Join(t.TempDir(), "missing-ca.pem")
			},
			secrets: []string{base64.StdEncoding.EncodeToString([]byte("id:do-not-log-api-key"))},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validConfig()
			test.configure(config)
			err := config.Validate()
			if test.validationError {
				require.Error(t, err, "invalid credentials must fail validation")
			} else {
				require.NoError(t, err, "valid credentials must pass structural validation")
				extension, createErr := createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
				require.Nil(t, extension, "TLS preflight must fail before extension construction")
				require.Error(t, createErr, "missing TLS material must fail construction")
				err = createErr
			}
			formatted := fmt.Sprintf("%+v", config)
			for _, secret := range test.secrets {
				require.NotContains(t, err.Error(), secret, "validation errors must not expose credentials")
				require.NotContains(t, formatted, secret, "formatted configuration must redact credentials")
			}
		})
	}
}

func validConfig() *Config {
	defaultConfig := createDefaultConfig()
	config, ok := defaultConfig.(*Config)
	if !ok {
		panic("elasticsearchauth default config has unexpected type")
	}
	return &Config{
		ClientConfig: config.ClientConfig,
		Endpoints:    []string{"https://es.example:9200"},
	}
}

func createTestExtension(t *testing.T, config *Config) *authenticator {
	t.Helper()
	extension, err := createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
	require.NoError(t, err, "extension creation must succeed")
	authenticator, ok := extension.(*authenticator)
	require.True(t, ok, "created extension must be authenticator")
	return authenticator
}

func generateCAAndCertificate(t *testing.T) (tls.Certificate, tls.Certificate) {
	t.Helper()
	ca, err := tlscommontest.GenCA()
	require.NoError(t, err, "CA generation must succeed")
	certificate, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "localhost", nil, []net.IP{net.IPv4(127, 0, 0, 1)}, false)
	require.NoError(t, err, "certificate generation must succeed")
	return ca, certificate
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
	return nil, fmt.Errorf("base transport must not be used")
}

type closeTrackingRoundTripper struct {
	closeCalls int
}

func (*closeTrackingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("round trip is not used by close test")
}

func (c *closeTrackingRoundTripper) CloseIdleConnections() {
	c.closeCalls++
}
