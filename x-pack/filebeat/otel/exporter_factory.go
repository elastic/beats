// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package otel provides Open Telemetry Exporters and Encoders. The package
// provides a global Exporter that is separate from the SDK global Exporter
// to allow inputs to share an Exporter that is separate from the application.
package otel

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// A global singleton factory. This is set by GetGlobalMetricsExporterFactory()
// This singleton can return a global Exporter that can be used by all the inputs.
var (
	exporterOnce    sync.Once
	exporterFactory *MetricsExporterFactory
)

type ExporterType string

const (
	GRPC    ExporterType = "grpc"
	HTTP    ExporterType = "http"
	console ExporterType = "console"
	None    ExporterType = "none"
)

// MetricExporterOptions are options used to create the Exporter.
type MetricExporterOptions struct {
	GrpcOptions    []otlpmetricgrpc.Option
	ConsoleOptions []stdoutmetric.Option
	HttpOptions    []otlpmetrichttp.Option
}

// GetDefaultMetricExporterOptions returns the default set of MetricExporterOptions.
func GetDefaultMetricExporterOptions() MetricExporterOptions {
	return MetricExporterOptions{
		GrpcOptions: []otlpmetricgrpc.Option{otlpmetricgrpc.WithTemporalitySelector(DeltaSelector)},
		ConsoleOptions: []stdoutmetric.Option{
			stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithTemporalitySelector(DeltaSelector),
			stdoutmetric.WithEncoder(&ConcurrentEncoder{Encoder: json.NewEncoder(os.Stdout)}),
		},
		HttpOptions: []otlpmetrichttp.Option{otlpmetrichttp.WithTemporalitySelector(DeltaSelector)},
	}
}

// The MetricsExporterFactory reads environment variables and creates the metrics exporter requested
// by the environment.
//
// To export OTEL metrics to an OTLP/gRPC endpoint set these environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT: Required. The OTLP endpoint URL.
//   - OTEL_EXPORTER_OTLP_HEADERS: Required if endpoint is authenticated.
//
// To export using the http/protobuf protocol add this environment variable with the above environment variables:
//   - OTEL_EXPORTER_OTLP_METRICS_PROTOCOL="http/protobuf"
//
// To override other environment variables and print a JSON representation of the metrics to console set:
//   - OTEL_METRICS_EXPORTER=console
//
// To override other environment variables and disable OTEL metrics collection set:
//   - OTEL_METRICS_EXPORTER=none
type MetricsExporterFactory struct {
	lock                  sync.Mutex
	globalMetricsExporter sdkmetric.Exporter
	exporterOptions       MetricExporterOptions
}

// initializeGlobalMetricsExporterFactory creates the global exporter factory
func initializeGlobalMetricsExporterFactory() {
	exporterFactory = NewMetricsExporterFactory(GetDefaultMetricExporterOptions())
}

// GetGlobalMetricsExporterFactory returns a globally defined MetricsExporterFactory.
// The GlobalExporterFactory returns the same Exporter for every call to
// GetExporter(ctx, true). The global MetricsExporterFactory is created using
// default MetricExporterOptions.
// Using the same Exporter across inputs reduces connections to the OTLP endpoint.
func GetGlobalMetricsExporterFactory() *MetricsExporterFactory {
	exporterOnce.Do(initializeGlobalMetricsExporterFactory) // Guarantees initializeConfig runs exactly once
	return exporterFactory
}

// NewMetricsExporterFactory creates a new MetricsExporterFactory
// exporterOptions MetricExporterOptions: the options to use for creating the Exporter
func NewMetricsExporterFactory(exporterOptions MetricExporterOptions) *MetricsExporterFactory {
	return &MetricsExporterFactory{
		globalMetricsExporter: nil,
		exporterOptions:       exporterOptions,
	}
}

// SetGlobalMetricsExporter sets the global metrics exporter.
// This is used for testing.
func (ef *MetricsExporterFactory) SetGlobalMetricsExporter(exporter sdkmetric.Exporter) {
	ef.lock.Lock()
	ef.globalMetricsExporter = exporter
	ef.lock.Unlock()
}

// GetExporter returns a metrics exporter based on the current environment
// configuration.
//
// If global is true, GetExporter returns the cached global exporter, creating it
// on first use. If global is false, GetExporter always creates and returns a new
// exporter.
func (ef *MetricsExporterFactory) GetExporter(ctx context.Context, global bool) (sdkmetric.Exporter, ExporterType, error) {
	exporterType := GetExporterTypeFromEnv()
	var err error

	if global {
		ef.lock.Lock()
	}

	defer func() {
		if global {
			ef.lock.Unlock()
		}
	}()

	if global && ef.globalMetricsExporter != nil {
		return ef.globalMetricsExporter, exporterType, nil
	}

	var exporter sdkmetric.Exporter
	switch exporterType {
	case console:
		exporter, err = stdoutmetric.New(ef.exporterOptions.ConsoleOptions...)
	case GRPC:
		exporter, err = otlpmetricgrpc.New(ctx, ef.exporterOptions.GrpcOptions...)

	case HTTP:
		exporter, err = otlpmetrichttp.New(ctx, ef.exporterOptions.HttpOptions...)

	default:
		exporter = nil
		err = nil
	}

	if global {
		ef.globalMetricsExporter = exporter
	}
	return exporter, exporterType, err
}

// DeltaSelector determines the temporality for a given instrument kind.
// All instruments use metricdata.DeltaTemporality
func DeltaSelector(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	// Using a switch because though the current implementation returns metricdata.DeltaTemporality
	// for all instruments, this may not be the case in the future.
	switch kind {
	case sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindGauge,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindObservableGauge,
		sdkmetric.InstrumentKindObservableCounter,
		sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter:
		return metricdata.DeltaTemporality
	}
	return metricdata.DeltaTemporality
}

// GetExporterTypeFromEnv determines the exporter type based on environment variables.
// Checks for OTEL_METRICS_EXPORTER and OTEL_EXPORTER_OTLP_METRICS_PROTOCOL to determine the exporter type.
// If OTEL_EXPORTER_OTLP_ENDPOINT is not set, no OTLP exporter will be generated regardless of the other
// environment variables.
// If no configuration is found, defaults to None.

func GetExporterTypeFromEnv() ExporterType {
	/*
		OTEL_METRICS_EXPORTER are:

		"otlp": OTLP
		"prometheus": Prometheus
		"console": Standard Output
		"logging": Standard Output. It is a deprecated value left for backwards compatibility. It SHOULD NOT be supported by new implementations.
		"none": No automatically configured exporter for metrics.

		OTEL_EXPORTER_OTLP_METRICS_PROTOCOL
			grpc to use OTLP/gRPC
			http/protobuf to use OTLP/HTTP + protobuf
			http/json to use OTLP/HTTP + JSON (not available in golang)
	*/

	// this is the expected setup for agentless
	exporter, ok := os.LookupEnv("OTEL_METRICS_EXPORTER")
	if ok && exporter == "console" {
		return console
	}
	if ok && exporter == "none" {
		return None
	}
	if ok && exporter == "prometheus" {
		return None
	}

	_, ok = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if ok {
		protocol, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
		if !ok {
			return GRPC
		}
		if strings.Contains(strings.ToLower(protocol), string(GRPC)) {
			return GRPC
		}
		if strings.ToLower(protocol) == "http/protobuf" {
			return HTTP
		}
	}
	return None
}
