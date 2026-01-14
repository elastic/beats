// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"unicode"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracerProviderMu sync.Mutex
	tracerProvider   *sdktrace.TracerProvider
)

// GetGlobalTracerProvider returns an existing or new global TracerProvider
func GetGlobalTracerProvider(ctx context.Context, resourceAttributes []attribute.KeyValue) (*sdktrace.TracerProvider, error) {
	tracerProviderMu.Lock()
	defer tracerProviderMu.Unlock()

	if tracerProvider == nil {
		tp, err := newTracerProvider(ctx, resourceAttributes)
		if err != nil {
			return nil, err
		}
		otel.SetTracerProvider(tp)
		tracerProvider = tp
	}

	return tracerProvider, nil
}

func newTracerProvider(ctx context.Context, resourceAttributes []attribute.KeyValue) (*sdktrace.TracerProvider, error) {
	// Make "none" the default exporter (rather than "oltp")
	const otelTracesExporterKey = "OTEL_TRACES_EXPORTER"
	if _, ok := os.LookupEnv(otelTracesExporterKey); !ok {
		os.Setenv(otelTracesExporterKey, "none")
	}

	// New exporter based on OTEL_TRACES_EXPORTER and OTEL_EXPORTER_OTLP_PROTOCOL
	// TODO probably switch away from autoexport later to avoid unnecessary dependencies
	exp, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	// Create a resource with attributes from various sources
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

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)),
		sdktrace.WithResource(res),
	)

	return tp, nil
}

var _ http.RoundTripper = (*ExtraSpanAttribsRoundTripper)(nil)

func NewExtraSpanAttribsRoundTripper(next http.RoundTripper) *ExtraSpanAttribsRoundTripper {
	return &ExtraSpanAttribsRoundTripper{
		next: next,
	}
}

type ExtraSpanAttribsRoundTripper struct {
	next http.RoundTripper
}

func (rt ExtraSpanAttribsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	span := trace.SpanFromContext(r.Context())
	if span != nil && span.SpanContext().IsValid() {
		for h := range r.Header {
			addHeaderAttr(span, "http.request.header.", h, r.Header)
		}
	}

	resp, err := rt.next.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	if span != nil && span.SpanContext().IsValid() && resp != nil {
		for h := range resp.Header {
			addHeaderAttr(span, "http.response.header.", h, resp.Header)
		}
	}

	return resp, nil
}

func addHeaderAttr(span trace.Span, prefix string, name string, headers http.Header) {
	const maxVals = 10
	const maxValLen = 1024

	values := headers.Values(name)
	if values == nil {
		return
	}
	if SensitiveName(name) {
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
// word that suggests it may contain secret data.
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
	"access":        {},
	"apikey":        {},
	"assertion":     {},
	"auth":          {},
	"authorization": {},
	"bearer":        {},
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
