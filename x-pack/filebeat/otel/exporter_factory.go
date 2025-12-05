// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type GetExporter interface {
	GetExporter(ctx context.Context) (sdkmetric.Exporter, error)
	GetType() ExporterType
}
type ExporterType string

const (
	GRPC    ExporterType = "grpc"
	HTTP    ExporterType = "http"
	console ExporterType = "console"
	None    ExporterType = "none"
)

var exporterFactory *ExporterFactory

type ExporterFactory struct {
	global                bool
	globalMetricsExporter sdkmetric.Exporter
	log                   *logp.Logger
	grpcOptions           []otlpmetricgrpc.Option
	consoleOptions        []stdoutmetric.Option
	httpOptions           []otlpmetrichttp.Option
}

var GlobalFactoryLock sync.Mutex

func GetGlobalExporterFactory(log *logp.Logger) *ExporterFactory {
	GlobalFactoryLock.Lock() // Acquire the lock
	defer GlobalFactoryLock.Unlock()
	if exporterFactory == nil {
		exporterFactory = NewExporterFactory(log, true)
	}
	return exporterFactory
}

// NewExporterFactory creates a new ExporterFactory with default configurations for GRPC, HTTP, and Console exporters.
//
// The factory will cache the created exporter to avoid recreating it multiple times.
//
// Args:
//
//	log (*logp.Logger): A logger instance used for logging within the factory.
//
// Returns:
//
//	*ExporterFactory: A new ExporterFactory instance with default configurations.
func NewExporterFactory(log *logp.Logger, global bool) *ExporterFactory {
	return &ExporterFactory{global: global, globalMetricsExporter: nil, log: log,
		grpcOptions: []otlpmetricgrpc.Option{otlpmetricgrpc.WithTemporalitySelector(DeltaSelector)},
		consoleOptions: []stdoutmetric.Option{stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithTemporalitySelector(DeltaSelector),
			stdoutmetric.WithEncoder(NewConcurentEncoder(json.NewEncoder(os.Stdout)))},
		httpOptions: []otlpmetrichttp.Option{otlpmetrichttp.WithTemporalitySelector(DeltaSelector)},
	}
}

// NewExporter creates a new exporter based on exporter type which is determined from the environment
// Exporters are not required to be concurrency safe. This factory creates a new exporter
// for each call to GetExporter
// Args:
//     ctx (context.Context): A context object created using the Go standard library.
//
// Returns:
//     sdkmetric.Exporter: A new exporter instance of the sdkmetric.Exporter type, along with an error if creation fails.

func (ef *ExporterFactory) GetExporter(ctx context.Context) (sdkmetric.Exporter, ExporterType, error) {

	exporterType := GetExporterTypeFromEnv()
	var err error
	exporter := ef.globalMetricsExporter
	if exporter == nil || !ef.global {

		switch exporterType {
		case console:
			exporter, err = stdoutmetric.New(ef.consoleOptions...)
		case GRPC:
			exporter, err = otlpmetricgrpc.New(ctx, ef.grpcOptions...)

		case HTTP:
			exporter, err = otlpmetrichttp.New(ctx, ef.httpOptions...)
		}
		if ef.global {
			ef.globalMetricsExporter = exporter
		}
	}
	return exporter, exporterType, err
}

// SetHttpOptions sets the HTTP options for the factory.
// Checks for equality before overwriting.
//
// Args:
//
//	options ([]otlpmetrichttp.Option): The new HTTP options to set.
func (ef *ExporterFactory) SetHttpOptions(options []otlpmetrichttp.Option) {
	if !slices.Equal(ef.httpOptions, options) {
		ef.httpOptions = options

	}
}

// SetGRPCOptions sets the gRPC options for the factory.
// Checks for equality before overwriting.
//
// Args:
//
//	options ([]otlpmetricgrpc.Option): The new gRPC options to set.
func (ef *ExporterFactory) SetGRPCOptions(options []otlpmetricgrpc.Option) {
	if !slices.Equal(ef.grpcOptions, options) {
		ef.grpcOptions = options

	}
}

// SetConsoleOptions sets the console options for the factory.
// Checks for equality before overwriting.
//
// Args:
//
//	options ([]stdoutmetric.Option): The new console options to set.
func (ef *ExporterFactory) SetConsoleOptions(options []stdoutmetric.Option) {
	if !slices.Equal(ef.consoleOptions, options) {
		ef.consoleOptions = options

	}
}

// DeltaSelector determines the temporality for a given instrument kind.
//
// # It returns metricdata.DeltaTemporality for all instruments
//
// Args:
//
//	kind (sdkmetric.InstrumentKind): The instrument kind to determine the temporality for.
//
// Returns:
//
//	metricdata.Temporality: The temporality determined for the given instrument kind.
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
//
// It checks for OTEL_METRICS_EXPORTER and OTEL_EXPORTER_OTLP_METRICS_PROTOCOL to determine the exporter type.
// If no configuration is found, it defaults to None.
//
// Args:
//     None
//
// Returns:
//     ExporterType: The exporter type determined from environment variables.

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
		if !ok || (ok && strings.Contains(strings.ToLower(protocol), string(GRPC))) {
			return GRPC
		}
		if ok && strings.ToLower(protocol) == "http/protobuf" {
			return HTTP
		}
	}
	return None

}
