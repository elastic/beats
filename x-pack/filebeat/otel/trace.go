// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func NewTracerProvider(ctx context.Context, resourceAttributes []attribute.KeyValue, inputName string) (*sdktrace.TracerProvider, error) {
	cfg, err := newExporterCfgFromEnv(inputName)
	if err != nil {
		return nil, err
	}
	exp, err := newExporterFromCfg(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(resourceAttributes...),
	)
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	if exp != nil {
		bsp := sdktrace.NewBatchSpanProcessor(exp)
		opts = append(opts, sdktrace.WithSpanProcessor(bsp))
	}

	return sdktrace.NewTracerProvider(opts...), nil
}

// newSpanExporterFromEnv creates a new exporter from configuration.
// It returns (nil, nil) if the exporter is "none".
func newExporterFromCfg(ctx context.Context, cfg *exporterCfg) (sdktrace.SpanExporter, error) {
	if cfg.disabled {
		return nil, nil
	}

	switch cfg.exporter {
	case "console":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp":
		switch cfg.protocol {
		case "grpc":
			var opts []otlptracegrpc.Option
			if cfg.endpoint != "" {
				opts = append(opts, otlptracegrpc.WithEndpoint(cfg.endpoint))
			}
			if len(cfg.headers) > 0 {
				opts = append(opts, otlptracegrpc.WithHeaders(cfg.headers))
			}
			if cfg.timeout > 0 {
				opts = append(opts, otlptracegrpc.WithTimeout(cfg.timeout))
			}
			if cfg.insecure {
				opts = append(opts, otlptracegrpc.WithInsecure())
			}
			return otlptracegrpc.New(ctx, opts...)
		case "http/protobuf":
			var opts []otlptracehttp.Option
			if cfg.endpoint != "" {
				opts = append(opts, otlptracehttp.WithEndpoint(cfg.endpoint))
			}
			if len(cfg.headers) > 0 {
				opts = append(opts, otlptracehttp.WithHeaders(cfg.headers))
			}
			if cfg.timeout > 0 {
				opts = append(opts, otlptracehttp.WithTimeout(cfg.timeout))
			}
			if cfg.insecure {
				opts = append(opts, otlptracehttp.WithInsecure())
			}
			return otlptracehttp.New(ctx, opts...)
		default:
			return nil, fmt.Errorf("unsupported OTLP traces protocol %q (expected grpc or http/protobuf)", cfg.protocol)
		}
	}

	return nil, nil
}

type exporterCfg struct {
	disabled bool
	exporter string
	protocol string
	endpoint string
	headers  map[string]string
	timeout  time.Duration
	insecure bool
}

// newExporterCfgFromEnv loads exporter configuration data from standard environment variables, in a form ready to use for exporter creation.
// The environment variables considered are:
// - BEATS_OTEL_TRACES_DISABLE (CSV values: cel,httpjson, default: (none))
// - OTEL_TRACES_EXPORTER (CSV values: none,otlp,console, first supported wins, default: none)
// - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL / OTEL_EXPORTER_OTLP_PROTOCOL (values: grpc|http/protobuf, default: grpc)
// - OTEL_EXPORTER_OTLP_TRACES_ENDPOINT / OTEL_EXPORTER_OTLP_ENDPOINT (e.g. "http://otlp-receiver.example.com:4317")
// - OTEL_EXPORTER_OTLP_TRACES_HEADERS  / OTEL_EXPORTER_OTLP_HEADERS  (e.g. "Authorization=Bearer abc123,X-Client-Version=1.2.3")
// - OTEL_EXPORTER_OTLP_TRACES_TIMEOUT  / OTEL_EXPORTER_OTLP_TIMEOUT  (in ms)
// - OTEL_EXPORTER_OTLP_TRACES_INSECURE / OTEL_EXPORTER_OTLP_INSECURE (values: true|false, default: true if http scheme used, otherwise false)
func newExporterCfgFromEnv(inputName string) (*exporterCfg, error) {
	cfg := exporterCfg{}

	rawDisable := strings.TrimSpace(os.Getenv("BEATS_OTEL_TRACES_DISABLE"))
	for _, disabledInput := range splitCSV(rawDisable) {
		if disabledInput == inputName {
			cfg.disabled = true
		}
	}

	rawExporter := strings.TrimSpace(os.Getenv("OTEL_TRACES_EXPORTER"))
	for _, rawName := range splitCSV(rawExporter) {
		name := strings.ToLower(rawName)
		switch name {
		case "none", "otlp", "console":
			cfg.exporter = name
		}
		if cfg.exporter != "" {
			break
		}
	}
	if rawExporter != "" && cfg.exporter == "" {
		return nil, fmt.Errorf("only unsupported trace exporter(s) found in OTEL_TRACES_EXPORTER=%q (supported: none, otlp, console)", rawExporter)
	}
	if cfg.exporter == "" {
		cfg.exporter = "none" // default
	}

	cfg.protocol = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"))
	if cfg.protocol == "" {
		cfg.protocol = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))
	}
	if cfg.protocol == "" {
		cfg.protocol = "grpc" // default
	}
	cfg.protocol = strings.ToLower(cfg.protocol)

	var err error
	var hasInsecure bool
	cfg.insecure, hasInsecure, err = envBoolFirstFound(
		"OTEL_EXPORTER_OTLP_TRACES_INSECURE",
		"OTEL_EXPORTER_OTLP_INSECURE",
	)
	if err != nil {
		return nil, err
	}

	cfg.endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
	if cfg.endpoint == "" {
		cfg.endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}
	u, err := url.Parse(cfg.endpoint)
	if err == nil && u.Host != "" {
		if u.Scheme == "http" && !hasInsecure {
			// Using a scheme of http rather than https indicates it will be insecure.
			cfg.insecure = true
		}
		// The endpoint was a URL like http://localhost:4318 but the exporter wants localhost:4318.
		cfg.endpoint = u.Host
	}

	headersStr := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS"))
	if headersStr == "" {
		headersStr = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))
	}
	cfg.headers, err = parseOTLPHeaders(headersStr)
	if err != nil {
		return nil, err
	}

	cfg.timeout = envDurationMillis("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT")
	if cfg.timeout == 0 {
		cfg.timeout = envDurationMillis("OTEL_EXPORTER_OTLP_TIMEOUT")
	}
	// 0 means "use exporter default"

	return &cfg, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseOTLPHeaders parses `key=value,key2=value2` into a map.
func parseOTLPHeaders(s string) (map[string]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	m := make(map[string]string)
	for _, part := range splitCSV(s) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid OTLP headers entry %q (expected key=value)", part)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			return nil, fmt.Errorf("invalid OTLP headers entry %q (empty key)", part)
		}
		m[k] = v
	}
	return m, nil
}

func envDurationMillis(key string) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	if n < 0 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}

func envBoolFirstFound(keys ...string) (val bool, found bool, err error) {
	for _, k := range keys {
		raw, ok := os.LookupEnv(k)
		if !ok {
			continue
		}
		found = true
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return false, true, fmt.Errorf("%s must be boolean: %w", k, err)
		}
		return b, true, nil
	}
	return false, false, nil
}

type TraceConfig struct {
	// Redacted is a list of headers and query string parameters that should have their values redacted in span attributes.
	Redacted []string `config:"redacted"`
	// Unredacted is a list of headers and query string parameters that should not have their values redacted in span attributes.
	Unredacted []string `config:"unredacted"`
}

// redactionReplacement is a string that can replace redacted values.
// Uses no characters that would require encoding in a URL.
const redactionReplacement = "REDACTED"

var _ http.RoundTripper = (*ExtraSpanAttribsRoundTripper)(nil)

func NewExtraSpanAttribsRoundTripper(next http.RoundTripper, traceCfg *TraceConfig) *ExtraSpanAttribsRoundTripper {
	redacted := make(map[string]bool)
	unredacted := make(map[string]bool)
	if traceCfg != nil {
		for _, name := range traceCfg.Redacted {
			redacted[strings.ToLower(name)] = true
		}
		for _, name := range traceCfg.Unredacted {
			unredacted[strings.ToLower(name)] = true
		}
	}
	return &ExtraSpanAttribsRoundTripper{
		next:       next,
		redacted:   redacted,
		unredacted: unredacted,
	}
}

type ExtraSpanAttribsRoundTripper struct {
	next       http.RoundTripper
	redacted   map[string]bool
	unredacted map[string]bool
}

func (rt ExtraSpanAttribsRoundTripper) shouldRedact(name string) bool {
	key := strings.ToLower(name)
	return rt.redacted[key] || (SensitiveName(name) && !rt.unredacted[key])
}

func (rt ExtraSpanAttribsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	span := trace.SpanFromContext(r.Context())
	if span != nil && span.SpanContext().IsValid() {
		for h := range r.Header {
			addHeaderAttr(span, "http.request.header.", h, r.Header, rt.shouldRedact)
		}
	}

	resp, err := rt.next.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	if span != nil && span.SpanContext().IsValid() {
		if resp != nil {
			for h := range resp.Header {
				addHeaderAttr(span, "http.response.header.", h, resp.Header, rt.shouldRedact)
			}
		}

		span.SetAttributes(attribute.StringSlice(
			"url.full",
			[]string{sanitizedURLString(r.URL, rt.shouldRedact)},
		))
	}

	return resp, nil
}

func sanitizedURLString(u *url.URL, shouldRedact func(string) bool) string {
	if u.RawQuery == "" {
		return u.String()
	}
	sanitized := *u
	sanitized.RawQuery = redactRawQuery(u.RawQuery, shouldRedact)
	return sanitized.String()
}

func redactRawQuery(raw string, shouldRedact func(name string) bool) string {
	replacementEnc := url.QueryEscape(redactionReplacement)

	parts := strings.Split(raw, "&")
	for i, part := range parts {
		if part == "" {
			continue
		}

		nameEnc, _, hasEq := strings.Cut(part, "=")
		if hasEq {
			name, err := url.QueryUnescape(nameEnc)
			if err == nil && shouldRedact(name) {
				parts[i] = nameEnc + "=" + replacementEnc
			}
		}
	}

	return strings.Join(parts, "&")
}

func addHeaderAttr(span trace.Span, prefix string, name string, headers http.Header, shouldRedact func(string) bool) {
	const maxVals = 10
	const maxValLen = 1024

	values := headers.Values(name)
	if values == nil {
		return
	}
	if shouldRedact(name) {
		values = []string{"REDACTED"}
	}
	if len(values) > maxVals {
		values = values[:maxVals]
	}
	for i, v := range values {
		if len(v) > maxValLen {
			values[i] = v[:maxValLen]
		}
	}

	key := prefix + strings.ToLower(name)
	span.SetAttributes(attribute.StringSlice(key, values))
}

// SensitiveName returns true if the given header or parameter name includes a
// word that suggests it may contain secret data. This is the default redaction
// logic.
func SensitiveName(name string) bool {
	words := splitToWords(name)
	for _, word := range words {
		if _, ok := sensitiveWords[word]; ok {
			return true
		}
	}
	return false
}

var sensitiveWords = map[string]struct{}{
	"apikey":        {},
	"assertion":     {},
	"auth":          {},
	"authorization": {},
	"code":          {},
	"cookie":        {},
	"credential":    {},
	"creds":         {},
	"hmac":          {},
	"jwt":           {},
	"key":           {},
	"nonce":         {},
	"passphrase":    {},
	"passwd":        {},
	"password":      {},
	"pwd":           {},
	"saml":          {},
	"secret":        {},
	"session":       {},
	"sid":           {},
	"sig":           {},
	"signature":     {},
	"sso":           {},
	"token":         {},
}

// splitToWords splits a string into words.
// It splits on separators, digit boundaries and case boundaries.
// The returned words are normalized to lowercase.
func splitToWords(s string) (words []string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return words
	}

	// common separators
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ':' || r == '/' || unicode.IsSpace(r)
	})

	for _, p := range parts {
		rs := []rune(p)
		if len(rs) == 0 {
			continue
		}

		start := 0
		emit := func(i int) {
			if i > start {
				words = append(words, strings.ToLower(string(rs[start:i])))
				start = i
			}
		}

		for i := 1; i < len(rs); i++ {
			a, b := rs[i-1], rs[i]
			var c rune
			if i+1 < len(rs) {
				c = rs[i+1]
			}

			// letter-digit boundary
			if (unicode.IsLetter(a) && unicode.IsDigit(b)) || (unicode.IsDigit(a) && unicode.IsLetter(b)) {
				emit(i)
				continue
			}

			// case boundary: camelCase => camel Case
			if unicode.IsLower(a) && unicode.IsUpper(b) {
				emit(i)
				continue
			}

			// case boundary, acronym: APIKey => API Key
			if unicode.IsUpper(a) && unicode.IsUpper(b) && c != 0 && unicode.IsLower(c) {
				emit(i)
				continue
			}
		}

		emit(len(rs))
	}
	return words
}
