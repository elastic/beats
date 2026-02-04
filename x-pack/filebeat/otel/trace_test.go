// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewExporterFromCfg_DisabledOverridesConfiguredExporter(t *testing.T) {
	cfg := &ExporterCfg{
		Disabled: true,
		Exporter: "console",
	}

	exp, err := newExporterFromCfg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp != nil {
		t.Fatal("expected exporter to be nil when disabled")
	}
}

func TestNewExporterCfgFromEnv_DisableWorksForCel(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "console")
	t.Setenv("BEATS_OTEL_TRACES_DISABLE", "foo,cel")

	cfg, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Disabled != true {
		t.Fatal("expected exporter to be disabled")
	}
}

func TestNewExporterCfgFromEnv_ExporterDefaultsToNone(t *testing.T) {
	// unset BEATS_OTEL_TRACES_DISABLE
	cfg, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Exporter != "none" {
		t.Fatalf("expected default exporter to be 'none', got %s", cfg.Exporter)
	}
}

func TestNewExporterCfgFromEnv_ExporterIsFirstAvaialble(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "unknown,console,otlp")
	cfg, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Exporter != "console" {
		t.Fatalf("expected default exporter to be 'none', got %s", cfg.Exporter)
	}
}

func TestNewExporterCfgFromEnv_ErrorForOnlyUnknonwExporters(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "foo,bar")
	cfg, err := newExporterCfgFromEnv("cel")
	if err == nil {
		t.Fatalf("expected error for configuring only unknown exporters. parsed config: %v", cfg)
	}
}

func TestNewExporterCfgFromEnv_ReadsGeneralOTLPVars(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp-receiver.example.com:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Bearer abc123,X-Client-Version=1.2.3")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "5000")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	got, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &ExporterCfg{
		Disabled:    false,
		Exporter:    "otlp",
		Protocol:    "http/protobuf",
		EndpointURL: "https://otlp-receiver.example.com:4317/v1/traces",
		Headers:     map[string]string{"Authorization": "Bearer abc123", "X-Client-Version": "1.2.3"},
		Timeout:     5 * time.Second,
		Insecure:    true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parsed exporter configuration mismatch (-want +got):\n%s", diff)
	}
}

func TestNewExporterCfgFromEnv_PrefersTraceSpecificOTLPVars(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp-receiver.example.com:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Bearer abc123,X-Client-Version=1.2.3")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "5000")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "grpc")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://otlp-receiver4.example.com:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "Authorization=Bearer abc124,X-Client-Version=1.2.4")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT", "4000")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "false")
	want := &ExporterCfg{
		Disabled:    false,
		Exporter:    "otlp",
		Protocol:    "grpc",
		EndpointURL: "http://otlp-receiver4.example.com:4317/v1/traces",
		Headers:     map[string]string{"Authorization": "Bearer abc124", "X-Client-Version": "1.2.4"},
		Timeout:     4 * time.Second,
		Insecure:    false,
	}
	got, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parsed exporter configuration mismatch (-want +got):\n%s", diff)
	}
}

func TestNewExporterCfgFromEnv_EndpointURLDefaultSchemeAndPath(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otlp-receiver.example.com:4317")
	got, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got.EndpointURL, "https://") {
		t.Fatalf("expected default schema of https to be added, got EndpointURL %s", got.EndpointURL)
	}
	if !strings.HasSuffix(got.EndpointURL, "/v1/traces") {
		t.Fatalf("expected default path of /v1/traces to be added, got EndpointURL %s", got.EndpointURL)
	}
}

func TestNewExporterCfgFromEnv_TracesEndpointURLDefaultSchemeAndPath(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "otlp-receiver.example.com:4317")
	got, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got.EndpointURL, "https://") {
		t.Fatalf("expected default schema of https to be added, got EndpointURL %s", got.EndpointURL)
	}
	if !strings.HasSuffix(got.EndpointURL, "/v1/traces") {
		t.Fatalf("expected default path of /v1/traces to be added, got EndpointURL %s", got.EndpointURL)
	}
}

func TestNewExporterCfgFromEnv_NotInsecureByDefault(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otlp-receiver.example.com:4317")
	// unset OTEL_EXPORTER_OTLP_INSECURE
	cfg, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Insecure {
		t.Fatal("expected insecure to be false by default")
	}
}

func TestNewExporterCfgFromEnv_SetsInsecureIfUnsetButUsingSchemeOfHTTP(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otlp-receiver.example.com:4317")
	// unset OTEL_EXPORTER_OTLP_INSECURE
	cfg, err := newExporterCfgFromEnv("cel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Insecure {
		t.Fatal("expected insecure to be true for HTTP")
	}
}

func TestSplitToWords(t *testing.T) {
	tcs := []struct {
		in   string
		want []string
	}{
		{"X-API-Key", []string{"x", "api", "key"}},
		{"APIToken", []string{"api", "token"}},
		{"sessionId", []string{"session", "id"}},
		{"userID1", []string{"user", "id", "1"}},
		{" X  YYY _ Token ", []string{"x", "yyy", "token"}},
		{"", nil},
	}

	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			got := splitToWords(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitToWords(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestSensitiveName(t *testing.T) {
	tcs := []struct {
		name string
		want bool
	}{
		{"Api-Key", true},
		{"Authorization", true},
		{"Cookie", true},
		{"Id-Token", true},
		{"Jwt", true},
		{"Session-Id", true},
		{"Set-Cookie", true},
		{"Signature", true},
		{"X-Access-Token", true},
		{"X-Apikey", true},
		{"X-Api-Key", true},
		{"X-Api-Token", true},
		{"X-Auth-Assertion", true},
		{"X-Auth-Credentials", true},
		{"X-Authorization-Signature", true},
		{"X-Auth-Session", true},
		{"X-Auth-Token", true},
		{"X-Client-Key", true},
		{"X-Credentials", true},
		{"X-Csrf-Token", true},
		{"X-Hmac-Signature", true},
		{"X-Identity-Assertion", true},
		{"X-Id-Token", true},
		{"X-Jwt", true},
		{"X-Jwt-Token", true},
		{"X-Nonce", true},
		{"X-Oauth-Access-Token", true},
		{"X-Oauth-Refresh-Token", true},
		{"X-Oauth-Token", true},
		{"X-Openid-Token", true},
		{"X-Passphrase", true},
		{"X-Password", true},
		{"X-Private-Key", true},
		{"X-Public-Key", true},
		{"X-Refresh-Token", true},
		{"X-Request-Nonce", true},
		{"X-Request-Signature", true},
		{"X-Saml-Assertion", true},
		{"X-Saml-Token", true},
		{"X-Session", true},
		{"X-Session-Id", true},
		{"X-Session-Token", true},
		{"X-Sid", true},
		{"X-Signature", true},
		{"X-Signature-Hmac", true},
		{"X-Signature-Timestamp", true},
		{"X-Signature-Version", true},
		{"X-Sso-Session", true},
		{"X-Sso-Token", true},
		{"X-User-Credentials", true},
		{"X-User-Session", true},
		{"X-Xsrf-Token", true},
		{"access_token", true},
		{"api_key", true},
		{"apikey", true},
		{"apiKey", true},
		{"assertion", true},
		{"authorization_code", true},
		{"auth_session", true},
		{"client_key", true},
		{"client_secret", true},
		{"code", true},
		{"credential", true},
		{"credentials", true},
		{"creds", true},
		{"hmac", true},
		{"hmac_signature", true},
		{"id_token", true},
		{"key", true},
		{"nonce", true},
		{"passphrase", true},
		{"passwd", true},
		{"password", true},
		{"private_key", true},
		{"public_key", true},
		{"pwd", true},
		{"refresh_token", true},
		{"saml_assertion", true},
		{"SAMLRequest", true},
		{"SAMLResponse", true},
		{"session", true},
		{"session_id", true},
		{"session_token", true},
		{"sid", true},
		{"sig", true},
		{"signature", true},
		{"sso_session", true},
		{"sso_token", true},
		{"token", true},
		{"token_type", true},

		{"client_id", false},
		{"limit", false},
		{"page", false},
		{"redirect_uri", false},
		{"signed", false},
		{"signed_request", false},
		{"sort_by", false},
		{"state", false},
		{"timestamp", false},
		{"user", false},
		{"username", false},
		{"X-Apitoken", false},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("SensitiveName(%q)==%v", tc.name, tc.want), func(t *testing.T) {
			got := SensitiveName(tc.name)
			if got != tc.want {
				t.Fatalf("SensitiveName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestShouldRedact(t *testing.T) {
	tcs := []struct {
		name string
		want bool
	}{
		// by default
		{"Authorization", true},
		{"apikey", true},
		{"Content-Type", false},
		{"timestamp", false},

		// overridden in trace config (input settings)
		{"X-Apitoken", true},
		{"state", true},
		{"Session-Id", false},
		{"code", false},

		// overridden in both directions - redacted wins
		{"username", true},
		{"password", true},
	}

	rt := NewExtraSpanAttribsRoundTripper(http.DefaultTransport, &TraceConfig{
		Redacted:   []string{"X-Apitoken", "state", "username", "password"},
		Unredacted: []string{"Session-Id", "code", "username", "password"},
	})

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("shouldRedact(%q)==%v", tc.name, tc.want), func(t *testing.T) {
			got := rt.shouldRedact(tc.name)
			if got != tc.want {
				t.Fatalf("shouldRedact(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestExtraSpanAttribsRoundTripper(t *testing.T) {
	ctx := context.Background()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	defer tp.Shutdown(ctx)

	rt := otelhttp.NewTransport(
		NewExtraSpanAttribsRoundTripper(
			roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Set-Cookie": []string{"secret"}},
					Body:       io.NopCloser(strings.NewReader("ok")),
				}, nil
			}),
			&TraceConfig{
				Redacted:   []string{"X-Special"},
				Unredacted: []string{"code"},
			},
		),
		otelhttp.WithTracerProvider(tp),
	)

	url := "http://example.com/foo?secret=secret&code=red"
	wantUrlFullAttr := "http://example.com/foo?secret=REDACTED&code=red"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Special", "secret")
	req.Header.Set("X-Page", "1")

	res, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	res.Body.Close()

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := spans[0]
	attrs := span.Attributes()

	authVals, ok := findAttr(attrs, attribute.Key("http.request.header.authorization"))
	if !ok || len(authVals) != 1 || authVals[0] != "REDACTED" {
		t.Fatalf("Authorization request header attr = %#v, want [REDACTED]", authVals)
	}
	xSpecialVals, ok := findAttr(attrs, attribute.Key("http.request.header.x-special"))
	if !ok || len(xSpecialVals) != 1 || xSpecialVals[0] != "REDACTED" {
		t.Fatalf("X-Special request header attr = %#v, want [REDACTED]", xSpecialVals)
	}
	pageVals, ok := findAttr(attrs, attribute.Key("http.request.header.x-page"))
	if !ok || len(pageVals) != 1 || pageVals[0] != "1" {
		t.Fatalf("X-Page request header attr = %#v, want [1]", pageVals)
	}
	setCookieVals, ok := findAttr(attrs, attribute.Key("http.response.header.set-cookie"))
	if !ok || len(setCookieVals) != 1 || setCookieVals[0] != "REDACTED" {
		t.Fatalf("Set-Cookie response header attr = %#v, want [REDACTED]", setCookieVals)
	}
	urlFullVals, ok := findAttr(attrs, attribute.Key("url.full"))
	if !ok || len(urlFullVals) != 1 || urlFullVals[0] != wantUrlFullAttr {
		t.Fatalf("url.full attr = %#v, want [%s]", urlFullVals, wantUrlFullAttr)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func findAttr(attrs []attribute.KeyValue, key attribute.Key) ([]string, bool) {
	for _, kv := range attrs {
		if kv.Key == key {
			if vals, ok := kv.Value.AsInterface().([]string); ok {
				return vals, true
			}
		}
	}
	return nil, false
}
