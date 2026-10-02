// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"gopkg.in/yaml.v3"
)

func TestConfigValidate(t *testing.T) {
	validAPIKey := configopaque.String("id:key")
	tests := []struct {
		name    string
		config  func() *Config
		errText string
	}{
		{
			name:   "valid unauthenticated destination",
			config: validConfig,
		},
		{
			name: "valid API key",
			config: func() *Config {
				config := validConfig()
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
			name: "valid authorization destination header",
			config: func() *Config {
				config := validConfig()
				config.Headers = configopaque.MapList{{Name: "Authorization", Value: "Bearer token"}}
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
			name: "API key without separator",
			config: func() *Config {
				config := validConfig()
				config.APIKey = "malformed-api-key"
				return config
			},
			errText: "raw non-empty id:key",
		},
		{
			name: "API key without ID",
			config: func() *Config {
				config := validConfig()
				config.APIKey = ":key"
				return config
			},
			errText: "raw non-empty id:key",
		},
		{
			name: "API key without key",
			config: func() *Config {
				config := validConfig()
				config.APIKey = "id:"
				return config
			},
			errText: "raw non-empty id:key",
		},
		{
			name: "base64-encoded API key",
			config: func() *Config {
				config := validConfig()
				config.APIKey = configopaque.String(base64.StdEncoding.EncodeToString([]byte("id:key")))
				return config
			},
			errText: "raw non-empty id:key",
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
			name: "API key and basic authentication conflict",
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
			name: "authorization header and API key conflict",
			config: func() *Config {
				config := validConfig()
				config.APIKey = validAPIKey
				config.Headers = configopaque.MapList{{Name: "authorization", Value: "Bearer token"}}
				return config
			},
			errText: "authorization header cannot be combined",
		},
		{
			name: "endpoint userinfo and basic authentication conflict",
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
			name: "endpoint userinfo and authorization header conflict",
			config: func() *Config {
				config := validConfig()
				config.Endpoints = []string{"https://url-user:url-password@es.example:9200"}
				config.Headers = configopaque.MapList{{Name: "Authorization", Value: "Bearer token"}}
				return config
			},
			errText: "userinfo cannot be combined",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config().Validate()
			if test.errText == "" {
				require.NoError(t, err, "valid configuration must pass validation")
				return
			}
			require.ErrorContains(t, err, test.errText, "invalid configuration must fail validation")
		})
	}
}

func TestConfigYAMLDecode(t *testing.T) {
	raw := decodeYAML(t, `
auth:
  authenticator: beatsauth/default
endpoints: [https://es.example:9200]
headers:
  Host: destination.example
  X-Extension: extension
user: elastic
password: password
`)

	//nolint:errcheck // It's a test, we know the type returned
	config := createDefaultConfig().(*Config)
	require.NoError(t, confmap.NewFromStringMap(raw).Unmarshal(config), "supported configuration must decode")
	require.NoError(t, config.Validate(), "decoded configuration must validate")
	require.True(t, config.Auth.HasValue(), "optional delegated auth must decode when configured")
	require.Equal(t, component.MustNewIDWithName("beatsauth", "default"), config.Auth.Get().AuthenticatorID, "authenticator component ID must decode")
	require.Equal(t, []string{"https://es.example:9200"}, config.Endpoints, "endpoints must decode")
	require.Equal(t, "elastic", config.User, "basic authentication user must decode")
	require.Equal(t, configopaque.String("password"), config.Password, "basic authentication password must decode")
	host, found := config.Headers.Get("Host")
	require.True(t, found, "destination Host header must decode")
	require.Equal(t, configopaque.String("destination.example"), host, "destination Host header value must decode")
}

func TestConfigYAMLDecodeWithoutAuth(t *testing.T) {
	raw := decodeYAML(t, `
endpoints: [https://es.example:9200]
user: elastic
password: password
headers:
  X-Extension: extension
`)

	//nolint:errcheck // It's a test, we know the type returned
	config := createDefaultConfig().(*Config)
	require.NoError(t, confmap.NewFromStringMap(raw).Unmarshal(config), "simple configuration must decode")
	require.NoError(t, config.Validate(), "simple configuration must validate without auth")
	require.False(t, config.Auth.HasValue(), "auth must remain optional when omitted")
}

func TestNestedAuthenticatorTransportComposition(t *testing.T) {
	base := &recordingRoundTripper{}
	nestedTransport := &nestedRoundTripper{}
	nestedAuth := &testHTTPClientAuthenticator{
		wrap: func(base http.RoundTripper) http.RoundTripper {
			nestedTransport.base = base
			return nestedTransport
		},
	}
	config := validConfig()
	config.Auth = configoptional.Some(configauth.Config{AuthenticatorID: nestedAuthenticatorID()})
	config.Headers = configopaque.MapList{{Name: "X-Extension", Value: "extension"}}
	authenticator := createTestExtension(t, config)

	require.NoError(t, authenticator.Start(t.Context(), extensionsHost{nestedAuthenticatorID(): nestedAuth}), "nested authenticator resolution must succeed")

	roundTripper, err := authenticator.RoundTripper(base)
	require.NoError(t, err, "nested transport composition must succeed")
	require.Same(t, base, nestedAuth.base, "nested authenticator must receive the consumer-supplied base transport")

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://es.example:9200", nil)
	require.NoError(t, err, "request creation must succeed")
	response, err := roundTripper.RoundTrip(request)
	require.NoError(t, err, "composed request must succeed")
	require.NoError(t, response.Body.Close(), "response body must close")
	require.Equal(t, "nested", base.request.Header.Get("X-Nested"), "nested transport must participate in the request")
	require.Equal(t, "extension", base.request.Header.Get("X-Extension"), "destination headers must wrap the nested transport")

	_, exposesClose := roundTripper.(interface{ CloseIdleConnections() })
	require.True(t, exposesClose, "elasticsearchauth wrapper must expose idle connection cleanup")
}

func TestRoundTripperClosesIdleConnections(t *testing.T) {
	base := &recordingRoundTripper{}
	authenticator := createTestExtension(t, validConfig())
	roundTripper, err := authenticator.RoundTripper(base)
	require.NoError(t, err, "simple mode must wrap the supplied base transport")

	closer, ok := roundTripper.(interface{ CloseIdleConnections() })
	require.True(t, ok, "elasticsearchauth wrapper must expose idle connection cleanup")
	closer.CloseIdleConnections()

	require.Equal(t, 1, base.closedIdleConnections, "idle connection cleanup must reach the wrapped transport")
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
				config.APIKey = "id:key"
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
			name: "unauthenticated destination",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := &recordingRoundTripper{}
			config := validConfig()
			config.Headers = configopaque.MapList{
				{Name: "Host", Value: "destination.example"},
				{Name: "X-Extension", Value: "extension"},
			}
			if test.configure != nil {
				test.configure(config)
			}
			authenticator := createTestExtension(t, config)
			roundTripper, err := authenticator.RoundTripper(base)
			require.NoError(t, err, "simple mode must wrap the supplied base transport without Start")

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://es.example:9200", nil)
			require.NoError(t, err, "request creation must succeed")
			request.Header.Set("X-Caller", "caller")
			response, err := roundTripper.RoundTrip(request)
			require.NoError(t, err, "authenticated request must succeed")
			require.NoError(t, response.Body.Close(), "response body must close")

			require.Equal(t, test.expectAuth, base.request.Header.Get("Authorization"), "configured Elasticsearch credentials must be applied")
			require.Equal(t, "extension", base.request.Header.Get("X-Extension"), "destination headers must be applied")
			require.Equal(t, "destination.example", base.request.Host, "destination Host override must be applied")
			require.Empty(t, request.Header.Get("Authorization"), "caller request must not gain authentication")
			require.Empty(t, request.Header.Get("X-Extension"), "caller request must not gain destination headers")
			require.Equal(t, "caller", request.Header.Get("X-Caller"), "caller headers must remain unchanged")
		})
	}
}

func TestNestedAuthenticatorResolutionErrors(t *testing.T) {
	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		errText    string
	}{
		{
			name:    "missing extension",
			errText: `failed to resolve authenticator "beatsauth/default": authenticator not found`,
		},
		{
			name: "wrong extension type",
			extensions: map[component.ID]component.Component{
				nestedAuthenticatorID(): &nonHTTPClientExtension{},
			},
			errText: "requested authenticator is not a HTTP client authenticator",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validConfig()
			config.Auth = configoptional.Some(configauth.Config{AuthenticatorID: nestedAuthenticatorID()})
			authenticator := createTestExtension(t, config)
			err := authenticator.Start(t.Context(), extensionsHost(test.extensions))
			require.ErrorContains(t, err, test.errText, "Start must report invalid nested authenticator configuration")
		})
	}
}

func TestRoundTripperDefaultsNilBaseWithoutNestedAuthenticator(t *testing.T) {
	base := &recordingRoundTripper{}
	originalDefaultTransport := http.DefaultTransport
	http.DefaultTransport = base
	t.Cleanup(func() {
		http.DefaultTransport = originalDefaultTransport
	})

	authenticator := createTestExtension(t, validConfig())
	roundTripper, err := authenticator.RoundTripper(nil)
	require.NoError(t, err, "simple mode must accept a nil base transport")

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://es.example:9200", nil)
	require.NoError(t, err, "request creation must succeed")
	response, err := roundTripper.RoundTrip(request)
	require.NoError(t, err, "request must use http.DefaultTransport")
	require.NoError(t, response.Body.Close(), "response body must close")
	require.NotNil(t, base.request, "http.DefaultTransport must receive the request")
	require.Equal(t, request.URL, base.request.URL, "http.DefaultTransport must receive the destination URL")
}

func TestEndpointsReturnsCopy(t *testing.T) {
	authenticator := createTestExtension(t, validConfig())
	endpoints := authenticator.Endpoints()
	endpoints[0] = "https://changed.example:9200"
	require.Equal(t, []string{"https://es.example:9200"}, authenticator.Endpoints(), "callers must not mutate configured endpoints")
}

func TestCredentialRedaction(t *testing.T) {
	config := validConfig()
	config.Password = "do-not-log-password"
	config.User = "elastic"
	config.Headers = configopaque.MapList{{Name: "X-Secret", Value: "do-not-log-header"}}
	formatted := fmt.Sprintf("%+v", config)
	require.NotContains(t, formatted, "do-not-log-password", "formatted configuration must redact passwords")
	require.NotContains(t, formatted, "do-not-log-header", "formatted configuration must redact destination headers")

	config.Password = ""
	config.User = ""
	config.APIKey = "do-not-log-malformed-api-key"
	err := config.Validate()
	require.Error(t, err, "malformed API key must fail validation")
	require.NotContains(t, err.Error(), "do-not-log-malformed-api-key", "validation error must not expose API keys")
}

func validConfig() *Config {
	return &Config{
		Endpoints: []string{"https://es.example:9200"},
	}
}

func nestedAuthenticatorID() component.ID {
	return component.MustNewIDWithName("beatsauth", "default")
}

func createTestExtension(t *testing.T, config *Config) *authenticator {
	t.Helper()
	created, err := createExtension(t.Context(), extension.Settings{ID: component.NewID(Type)}, config)
	require.NoError(t, err, "extension creation must succeed")
	authenticator, ok := created.(*authenticator)
	require.True(t, ok, "created extension must be authenticator")
	return authenticator
}

func decodeYAML(t *testing.T, value string) map[string]any {
	t.Helper()
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(value), &raw), "YAML fixture must decode")
	return raw
}

type extensionsHost map[component.ID]component.Component

func (h extensionsHost) GetExtensions() map[component.ID]component.Component {
	return h
}

type nonHTTPClientExtension struct{}

func (*nonHTTPClientExtension) Start(context.Context, component.Host) error {
	return nil
}

func (*nonHTTPClientExtension) Shutdown(context.Context) error {
	return nil
}

type testHTTPClientAuthenticator struct {
	base http.RoundTripper
	wrap func(http.RoundTripper) http.RoundTripper
}

func (*testHTTPClientAuthenticator) Start(context.Context, component.Host) error {
	return nil
}

func (*testHTTPClientAuthenticator) Shutdown(context.Context) error {
	return nil
}

func (a *testHTTPClientAuthenticator) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	a.base = base
	return a.wrap(base), nil
}

type nestedRoundTripper struct {
	base http.RoundTripper
}

func (n *nestedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("X-Nested", "nested")
	return n.base.RoundTrip(request)
}

type recordingRoundTripper struct {
	request               *http.Request
	closedIdleConnections int
}

func (r *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	r.request = request
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    request,
	}, nil
}

func (r *recordingRoundTripper) CloseIdleConnections() {
	r.closedIdleConnections++
}
