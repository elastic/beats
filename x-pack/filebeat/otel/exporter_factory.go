package otel

import (
	"context"
	"errors"
	"fmt"
	"github.com/elastic/elastic-agent-libs/logp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"os"
	"strconv"
	"strings"
	"time"
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

type CollectionPeriodType string

const (
	Manual   CollectionPeriodType = "manual"
	Fixed    CollectionPeriodType = "fixed"
	Interval CollectionPeriodType = "interval"
)

type HttpExporterConfig struct {
	options []otlpmetrichttp.Option
}

func (c HttpExporterConfig) GetExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	return otlpmetrichttp.New(ctx, c.options...)
}

func (c HttpExporterConfig) GetType() ExporterType {
	return HTTP
}

type GRPCExporterConfig struct {
	options []otlpmetricgrpc.Option
}

func (c GRPCExporterConfig) GetExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	return otlpmetricgrpc.New(ctx, c.options...)
}

func (c GRPCExporterConfig) GetType() ExporterType {
	return GRPC
}

type ConsoleExporterConfig struct {
	options []stdoutmetric.Option
}

func (c ConsoleExporterConfig) GetExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	return stdoutmetric.New(c.options...)
}

func (c ConsoleExporterConfig) GetType() ExporterType {
	return console
}

type ExporterFactory struct {
	log             *logp.Logger
	exporterGetters map[ExporterType]GetExporter
}

func NewExporterFactory(log *logp.Logger) *ExporterFactory {
	return &ExporterFactory{log: log, exporterGetters: map[ExporterType]GetExporter{}}
}
func (ef *ExporterFactory) NewExporter(ctx context.Context, exporterType ExporterType) (sdkmetric.Exporter, error) {
	if exporterType == None {
		return nil, nil
	}
	getter, ok := ef.exporterGetters[exporterType]
	if !ok {
		switch exporterType {
		case console:
			getter = ConsoleExporterConfig{
				options: []stdoutmetric.Option{stdoutmetric.WithPrettyPrint(), stdoutmetric.WithTemporalitySelector(DeltaSelector)},
			}
		case GRPC:
			getter = GRPCExporterConfig{
				options: []otlpmetricgrpc.Option{otlpmetricgrpc.WithTemporalitySelector(DeltaSelector)},
			}
		case HTTP:
			getter = HttpExporterConfig{
				options: []otlpmetrichttp.Option{otlpmetrichttp.WithTemporalitySelector(DeltaSelector)},
			}

		}
		if getter == nil {
			return nil, fmt.Errorf("unknown exporter type: %s", exporterType)
		}
	}
	return getter.GetExporter(ctx)
}

func (ef *ExporterFactory) SetHttpOptions(options []otlpmetrichttp.Option) {
	ef.exporterGetters[HTTP] = HttpExporterConfig{options: options}
}
func (ef *ExporterFactory) SetGRPCOptions(options []otlpmetricgrpc.Option) {
	ef.exporterGetters[GRPC] = GRPCExporterConfig{options: options}
}
func (ef *ExporterFactory) SetConsoleOptions(options []stdoutmetric.Option) {
	ef.exporterGetters[console] = ConsoleExporterConfig{options: options}
}

func DeltaSelector(kind sdkmetric.InstrumentKind) metricdata.Temporality {
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
	panic("unknown instrument kind")
}

/*
GetExporterTypeFromEnv()

	Checks environment for OTEL environment variables that are needed for APM opentelemetry.
	Defaults to OTLP/gRPC output when APM is configured.
	Supports debugging to console if APM is not configured. Set OTEL_METRICS_EXPORTER=console to export to console.
	Uses a noop metrics exporter if no OTEL environment variables are configured.
*/
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
	if IsOTLPExport() {
		protocol, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
		if ok && strings.Contains(protocol, string(HTTP)) {
			return HTTP
		}
		return GRPC
	}

	// we can also export to the console for debugging purposes
	exporter, ok := os.LookupEnv("OTEL_METRICS_EXPORTER")
	if ok && exporter == "console" {
		return console
	}
	// Allow the integration to start with no metrics.
	return None

}

func IsOTLPExport() bool {
	_, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		return false
	}
	_, ok = os.LookupEnv("OTEL_EXPORTER_OTLP_HEADERS")
	if !ok {
		return false
	}
	_, ok = os.LookupEnv("OTEL_RESOURCE_ATTRIBUTES")
	if !ok {
		return false
	}
	return true
}

func GetCollectionPeriodFromEnvironment(ctx context.Context, period time.Duration) (time.Duration, error) {
	collectionType, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_COLLECTION_PERIOD_TYPE")
	if !ok || strings.ToLower(collectionType) == string(Manual) {
		return 0, nil
	}
	if strings.ToLower(collectionType) == string(Fixed) {
		collectionInterval, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_COLLECTION_PERIOD_INTERVAL")
		if !ok || collectionInterval == "" {
			return 0, nil
		}
		collectionPeriod, err := strconv.Atoi(collectionInterval)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("OTEL metrics collection period type is Fixed, but interval in OTEL_EXPORTER_OTLP_METRICS_COLLECTION_PERIOD_INTERVAL %s is not an integer defined. Using manual metrics", collectionPeriod))
		}
	}
	if strings.ToLower(collectionType) == string(Interval) {
		return period, nil
	}
	return 0, errors.New(fmt.Sprintf("invalid OTEL metrics collection period type %s is unknown. Using manual metrics"))
}
