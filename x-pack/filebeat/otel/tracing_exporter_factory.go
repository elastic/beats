// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gofrs/uuid/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type TracesExporterType string

const (
	TracesGRPC    TracesExporterType = "grpc"
	TracesHTTP    TracesExporterType = "http"
	tracesDefault TracesExporterType = TracesGRPC
	serviceName                      = "filebeatreceiver"

	endpointEnvVar = "OTEL_EXPORTER_OTLP_ENDPOINT"
	protocolEnvVar = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
)

var (
	traceOnce      sync.Once
	traceShutdown  = func(context.Context) error { return nil }
	traceErr       error
	tracerProvider *sdktrace.TracerProvider
)

// newOTelTraceProvider builds an SDK TracerProvider if OTEL_TRACES_EXPORTER
// indicates traces should be exported.
func newOTelTraceProvider(ctx context.Context, version string) (*sdktrace.TracerProvider, error) {
	if !tracesEnabled() {
		return nil, nil
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(defaultServiceAttributes(version)...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build OTel resource: %w", err)
	}

	exporterType := tracesExporterType()
	exporter, err := buildExporter(ctx, exporterType)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter (%s): %w", exporterType, err)
	}

	spanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanProcessor),
		sdktrace.WithResource(res),
	)

	return traceProvider, nil
}

// buildExporter creates an OTLP trace exporter based on the given type (gRPC or HTTP).
func buildExporter(ctx context.Context, exporterType TracesExporterType) (sdktrace.SpanExporter, error) {
	var exporter sdktrace.SpanExporter
	var err error
	switch exporterType {
	case TracesHTTP:
		var option []otlptracehttp.Option
		// Allow skipping TLS verification for testing purposes with ELASTIC_OTEL_INSECURE_SKIP_VERIFY=true
		if v := os.Getenv("ELASTIC_OTEL_INSECURE_SKIP_VERIFY"); strings.ToLower(strings.TrimSpace(v)) == "true" {
			option = append(option, otlptracehttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}))
		}
		exporter, err = otlptracehttp.New(ctx, option...)
	case TracesGRPC:
		fallthrough
	default:
		exporter, err = otlptracegrpc.New(ctx)
	}

	return exporter, err
}

// tracesEnabled checks if tracing is enabled via OTEL_EXPORTER_OTLP_ENDPOINT.
func tracesEnabled() bool {
	endpoint, ok := os.LookupEnv(endpointEnvVar)

	return ok && strings.TrimSpace(endpoint) != ""
}

// tracesExporterType reads OTEL_EXPORTER_OTLP_TRACES_PROTOCOL to determine
func tracesExporterType() TracesExporterType {
	protocol, ok := os.LookupEnv(protocolEnvVar)
	if !ok || strings.TrimSpace(protocol) == "" {
		return tracesDefault
	}
	p := strings.ToLower(strings.TrimSpace(protocol))
	if strings.Contains(p, "http") {
		return TracesHTTP
	}
	if strings.Contains(p, "grpc") {
		return TracesGRPC
	}
	return tracesDefault
}

// defaultServiceAttributes returns default service.* attributes for the resource.
func defaultServiceAttributes(version string) []attribute.KeyValue {
	// Add a stable-ish service.instance.id if user didn't set it.
	inst := os.Getenv("OTEL_SERVICE_INSTANCE_ID")
	if strings.TrimSpace(inst) == "" {
		inst = uuid.Must(uuid.NewV4()).String()
	}

	return []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.ServiceInstanceID(inst),
	}
}

// TracerProvider configures the global OpenTelemetry tracer provider for
// the current process based on standard OTEL_* environment variables.
//
// It is safe to call multiple times.
func TracerProvider(ctx context.Context, version string) (*sdktrace.TracerProvider, error) {
	traceOnce.Do(func() {
		tracerProvider, traceErr = newOTelTraceProvider(ctx, version)
		if traceErr != nil || tracerProvider == nil {
			return
		}

		traceShutdown = func(ctx context.Context) error {
			// Ensure processor flushes.
			return tracerProvider.Shutdown(ctx)
		}
	})
	return tracerProvider, traceErr
}

// ShutdownTracing flushes and shuts down the global tracer provider configured
// by TracerProvider.
func ShutdownTracing(ctx context.Context) error {
	if traceShutdown == nil {
		return nil
	}
	return traceShutdown(ctx)
}
