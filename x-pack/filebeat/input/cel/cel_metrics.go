// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/x-pack/filebeat/otel"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

type inputMetrics struct {
	resource            *monitoring.String // URL-ish of input resource
	executions          *monitoring.Uint   // times the CEL program has been executed
	batchesReceived     *monitoring.Uint   // number of event arrays received
	eventsReceived      *monitoring.Uint   // number of events received
	batchesPublished    *monitoring.Uint   // number of event arrays published
	eventsPublished     *monitoring.Uint   // number of events published
	celProcessingTime   metrics.Sample     // histogram of the elapsed successful cel program processing times in nanoseconds
	batchProcessingTime metrics.Sample     // histogram of the elapsed successful batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches).
}

func newInputMetrics(reg *monitoring.Registry, logger *logp.Logger) (*inputMetrics, *monitoring.Registry) {
	out := &inputMetrics{
		resource:            monitoring.NewString(reg, "resource"),
		executions:          monitoring.NewUint(reg, "cel_executions"),
		batchesReceived:     monitoring.NewUint(reg, "batches_received_total"),
		eventsReceived:      monitoring.NewUint(reg, "events_received_total"),
		batchesPublished:    monitoring.NewUint(reg, "batches_published_total"),
		eventsPublished:     monitoring.NewUint(reg, "events_published_total"),
		celProcessingTime:   metrics.NewUniformSample(1024),
		batchProcessingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "cel_processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.celProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))

	return out, reg
}

type otelCELMetrics struct {
	log                                  *logp.Logger
	shutdownFuncs                        []func(context.Context) error
	manualExportFunc                     func(context.Context) error
	exportLock                           sync.Mutex
	export                               bool
	startRunTime                         time.Time
	periodicRunCount                     metric.Int64Counter
	periodicBatchProcessedCount          metric.Int64Counter
	periodicBatchPublishedCount          metric.Int64Counter
	periodicEventProcessedCount          metric.Int64Counter
	periodicEventPublishedCount          metric.Int64Counter
	periodicRunDuration                  metric.Float64Counter
	periodicCelDuration                  metric.Float64Counter
	periodicEventPublishDuration         metric.Float64Counter
	periodicProgramRunStartedCount       metric.Int64Counter
	periodicProgramRunSuccessCount       metric.Int64Counter
	programBatchProcessedHistogram       metric.Int64Histogram
	programBatchPublishedHistogram       metric.Int64Histogram
	programEventProcessedHistogram       metric.Int64Histogram
	programEventPublishedHistogram       metric.Int64Histogram
	programRunDurationHistogram          metric.Float64Histogram
	programCelDurationHistogram          metric.Float64Histogram
	programEventPublishDurationHistogram metric.Float64Histogram
}

// StartPeriodic starts the periodic metrics collection.
// exportLock blocks starting a new periodic if the
// export is still processing. This should not happen
// in the real world use due to the use of intervals for
// running periodic runs. However, test environments with
// small intervals could potentially cause this to happen.
func (o *otelCELMetrics) StartPeriodic(ctx context.Context) {
	o.exportLock.Lock()
	o.export = true
	o.exportLock.Unlock()
	o.periodicRunCount.Add(ctx, 1)
	o.startRunTime = time.Now()
}

// EndPeriodic ends the periodic metrics collection and manually exports metrics if a manual export function is set.
func (o *otelCELMetrics) EndPeriodic(ctx context.Context) {
	if o.export {
		o.periodicRunDuration.Add(ctx, time.Since(o.startRunTime).Seconds())
	}
	if o.manualExportFunc == nil || !o.export {
		return
	}
	o.exportLock.Lock() // Acquire the lock
	defer o.exportLock.Unlock()
	o.log.Debug("otelCELMetrics EndPeriodic called")
	o.export = false
	o.log.Debug("otelCELMetrics manual export export")

	err := o.manualExportFunc(ctx)
	if err != nil {
		o.log.Errorf("error exporting metrics: %v", err)
	}
	o.log.Debug("otelCELMetrics manual export ended")
}

func (o *otelCELMetrics) AddProgramRunDuration(ctx context.Context, duration time.Duration) {
	o.programRunDurationHistogram.Record(ctx, duration.Seconds())
}

func (o *otelCELMetrics) AddPublishDuration(ctx context.Context, duration time.Duration) {
	o.periodicEventPublishDuration.Add(ctx, duration.Seconds())
	o.programEventPublishDurationHistogram.Record(ctx, duration.Seconds())
}

func (o *otelCELMetrics) AddCELDuration(ctx context.Context, duration time.Duration) {
	o.periodicCelDuration.Add(ctx, duration.Seconds())
	o.programCelDurationHistogram.Record(ctx, duration.Seconds())
}

func (o *otelCELMetrics) AddReceivedBatch(ctx context.Context, count int64) {
	o.periodicBatchProcessedCount.Add(ctx, count)
	o.programBatchProcessedHistogram.Record(ctx, count)
}

func (o *otelCELMetrics) AddPublishedBatch(ctx context.Context, count int64) {
	o.periodicBatchPublishedCount.Add(ctx, count)
	o.programBatchPublishedHistogram.Record(ctx, count)
}

func (o *otelCELMetrics) AddReceivedEvents(ctx context.Context, count int64) {
	o.periodicEventProcessedCount.Add(ctx, count)
	o.programEventProcessedHistogram.Record(ctx, count)
}

func (o *otelCELMetrics) AddPublishedEvents(ctx context.Context, count int64) {
	o.periodicEventPublishedCount.Add(ctx, count)
	o.programEventPublishedHistogram.Record(ctx, count)
}

func (o *otelCELMetrics) AddProgramExecutionStarted(ctx context.Context, count int64) {
	o.periodicProgramRunStartedCount.Add(ctx, count)
}

func (o *otelCELMetrics) AddProgramExecutionSuccess(ctx context.Context, count int64) {
	o.periodicProgramRunSuccessCount.Add(ctx, count)
}

// Shutdown(ctx context.Context) error
func (o *otelCELMetrics) Shutdown(ctx context.Context) {
	o.EndPeriodic(ctx)
	// Consider adding an environment variable to control timeout time
	timeOutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var err error
	for _, fn := range o.shutdownFuncs {
		err = errors.Join(err, fn(timeOutContext))
	}
	if err != nil {
		o.log.Errorf("error shutting down metrics: %v", err)
	}
	cancel()
}

func newOTELCELMetrics(log *logp.Logger,
	resource resource.Resource,
	tripper http.RoundTripper,
	metricExporter sdkmetric.Exporter,
) (*otelCELMetrics, *otelhttp.Transport, error) {
	var manualExportFunc func(context.Context) error
	var meterProvider metric.MeterProvider
	var shutdownFuncs []func(context.Context) error
	if metricExporter == nil {
		meterProvider = noop.NewMeterProvider()
	} else {
		reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(otel.DeltaSelector))

		// By default, we force the use of base2_exponential_bucket_histogram for
		// efficiency. However, some backends (like Elastic APM Server) do not
		// support them yet. So we allow users to opt-out and use the default
		// explicit_bucket_histogram only by setting
		// OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION=explicit_bucket_histogram.
		//
		// Ref: https://opentelemetry.io/docs/specs/otel/metrics/sdk_exporters/otlp/
		var views []sdkmetric.View
		if os.Getenv("OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION") != "explicit_bucket_histogram" {
			exponentialView := sdkmetric.NewView(
				sdkmetric.Instrument{
					// captures every histogram that will produced by this provider
					Name: "*",
					Kind: sdkmetric.InstrumentKindHistogram,
				},
				sdkmetric.Stream{
					Aggregation: sdkmetric.AggregationBase2ExponentialHistogram{
						MaxSize:  160, // Optional: configure max buckets
						MaxScale: 20,  // Optional: configure max scale
					},
				},
			)
			views = []sdkmetric.View{exponentialView}
		}

		sdkMeterProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(&resource),
			sdkmetric.WithView(views...))
		shutdownFuncs = append(shutdownFuncs, sdkMeterProvider.Shutdown)
		meterProvider = sdkMeterProvider

		manualExportFunc = func(ctx context.Context) error {
			collectedMetrics := &metricdata.ResourceMetrics{}
			err := reader.Collect(ctx, collectedMetrics)
			if err != nil {
				return err
			}
			if log.IsDebug() {
				jsonData, err := json.Marshal(collectedMetrics)
				if err == nil {
					log.Debugf("otelCELMetrics Collected metrics %s", jsonData)
				} else {
					log.Debugf("otelCELMetrics could not marshall Collected metrics into json %v", collectedMetrics)
				}
			}
			go func(log *logp.Logger, metricExporter sdkmetric.Exporter, collectedMetrics *metricdata.ResourceMetrics) {
				timeOutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := metricExporter.Export(timeOutContext, collectedMetrics)
				if err != nil {
					log.Error("Failed to export metrics: ", err)
				}
				cancel()
			}(log, metricExporter, collectedMetrics)
			return nil
		}
	}
	transport := otelhttp.NewTransport(tripper, otelhttp.WithMeterProvider(meterProvider))

	meter := meterProvider.Meter("github.com/elastic/beats/x-pack/filebeat/otel/cel_metrics.go")

	periodicRunCount, err := meter.Int64Counter("input.cel.periodic.run",
		metric.WithDescription("Number of times a periodic run was started."),
		metric.WithUnit("{run}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.run: %w", err)
	}
	programRunStartedCount, err := meter.Int64Counter("input.cel.periodic.program.run.started",
		metric.WithDescription("Number of times a CEL program was started in a periodic run."),
		metric.WithUnit("{run}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.started: %w", err)
	}
	programRunSuccessCount, err := meter.Int64Counter("input.cel.periodic.program.run.success",
		metric.WithDescription("Number of times a CEL program terminated without an error in a periodic run."),
		metric.WithUnit("{run}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.success: %w", err)
	}
	periodicBatchCount, err := meter.Int64Counter("input.cel.periodic.batch.received",
		metric.WithDescription("Number of event batches generated in a periodic run."),
		metric.WithUnit("{batch}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.batch.received: %w", err)
	}
	periodicPublishedBatchCount, err := meter.Int64Counter("input.cel.periodic.batch.published",
		metric.WithDescription("Number of event batches successfully published in a periodic run."),
		metric.WithUnit("{batch}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.batch.published: %w", err)
	}
	periodicEventCount, err := meter.Int64Counter("input.cel.periodic.event.received",
		metric.WithDescription("Number of events generated in a periodic run."),
		metric.WithUnit("{event}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event.received: %w", err)
	}
	periodicPublishedEventCount, err := meter.Int64Counter("input.cel.periodic.event.published",
		metric.WithDescription("Number of events published in a periodic run."),
		metric.WithUnit("{event}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event.published: %w", err)
	}
	periodicTotalDuration, err := meter.Float64Counter("input.cel.periodic.run.duration",
		metric.WithDescription("Total time spent in a periodic run."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.run.duration: %w", err)
	}
	periodicCELDuration, err := meter.Float64Counter("input.cel.periodic.cel.duration",
		metric.WithDescription("Total time spent processing CEL programs in a periodic run."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.cel.duration: %w", err)
	}
	periodicPublishDuration, err := meter.Float64Counter("input.cel.periodic.event.publish.duration",
		metric.WithDescription("Total time spent publishing events in a periodic run."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event.publish.duration: %w", err)
	}

	programBatchProcessed, err := meter.Int64Histogram("input.cel.program.batch.received",
		metric.WithDescription("Number of event batches the CEL program has generated."),
		metric.WithUnit("{batch}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.received: %w", err)
	}
	programBatchPublished, err := meter.Int64Histogram("input.cel.program.batch.published",
		metric.WithDescription("Number of event batches the CEL program has published."),
		metric.WithUnit("{batch}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.published: %w", err)
	}
	programEventGenerated, err := meter.Int64Histogram("input.cel.program.event.received",
		metric.WithDescription("Number of events the CEL program has generated."),
		metric.WithUnit("{event}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.event.received: %w", err)
	}
	programEventPublished, err := meter.Int64Histogram("input.cel.program.event.published",
		metric.WithDescription("Number of events the CEL program has published."),
		metric.WithUnit("{event}"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.event.published: %w", err)
	}

	programRunDuration, err := meter.Float64Histogram("input.cel.program.run.duration",
		metric.WithDescription("Time spent executing the CEL program."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.duration: %w", err)
	}
	programCELDuration, err := meter.Float64Histogram("input.cel.program.cel.duration",
		metric.WithDescription("Time spent processing the CEL program."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.cel.duration: %w", err)
	}
	programPublishDuration, err := meter.Float64Histogram("input.cel.program.publish.duration",
		metric.WithDescription("Time spent publishing events in the CEL program."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.publish.duration: %w", err)
	}

	return &otelCELMetrics{
		log:                                  log,
		shutdownFuncs:                        shutdownFuncs,
		manualExportFunc:                     manualExportFunc,
		periodicRunCount:                     periodicRunCount,
		periodicBatchProcessedCount:          periodicBatchCount,
		periodicBatchPublishedCount:          periodicPublishedBatchCount,
		periodicEventProcessedCount:          periodicEventCount,
		periodicEventPublishedCount:          periodicPublishedEventCount,
		periodicRunDuration:                  periodicTotalDuration,
		periodicCelDuration:                  periodicCELDuration,
		periodicEventPublishDuration:         periodicPublishDuration,
		periodicProgramRunStartedCount:       programRunStartedCount,
		periodicProgramRunSuccessCount:       programRunSuccessCount,
		programBatchProcessedHistogram:       programBatchProcessed,
		programBatchPublishedHistogram:       programBatchPublished,
		programEventProcessedHistogram:       programEventGenerated,
		programEventPublishedHistogram:       programEventPublished,
		programRunDurationHistogram:          programRunDuration,
		programCelDurationHistogram:          programCELDuration,
		programEventPublishDurationHistogram: programPublishDuration,
	}, transport, nil
}

type metricsRecorder struct {
	inputMetrics *inputMetrics
	otelMetrics  *otelCELMetrics
}

func newMetricsRecorder(inputMetrics *inputMetrics, otelMetrics *otelCELMetrics) (*metricsRecorder, error) {
	if inputMetrics == nil || otelMetrics == nil {
		return nil, errors.New("input metrics and otel metrics cannot be nil")
	}
	return &metricsRecorder{
		inputMetrics,
		otelMetrics,
	}, nil
}

func (o *metricsRecorder) StartPeriodic(ctx context.Context) {
	o.otelMetrics.StartPeriodic(ctx)
}

// EndPeriodic ends the periodic metrics collection and manually exports metrics if a manual export function is set.
func (o *metricsRecorder) EndPeriodic(ctx context.Context) {
	o.otelMetrics.EndPeriodic(ctx)
}

func (o *metricsRecorder) AddCELDuration(ctx context.Context, duration time.Duration) {
	o.otelMetrics.AddCELDuration(ctx, duration)
	o.inputMetrics.celProcessingTime.Update(duration.Nanoseconds())
}

func (o *metricsRecorder) AddProgramRunDuration(ctx context.Context, duration time.Duration) {
	o.otelMetrics.AddProgramRunDuration(ctx, duration)
}

func (o *metricsRecorder) AddPublishDuration(ctx context.Context, duration time.Duration) {
	o.otelMetrics.AddPublishDuration(ctx, duration)
	o.inputMetrics.batchProcessingTime.Update(duration.Nanoseconds())
}

func (o *metricsRecorder) AddReceivedBatch(ctx context.Context, count uint) {
	o.inputMetrics.batchesReceived.Add(uint64(count))
	o.otelMetrics.AddReceivedBatch(ctx, int64(count)) //nolint:gosec // disable G115
}

func (o *metricsRecorder) AddPublishedBatch(ctx context.Context, count uint) {
	o.inputMetrics.batchesPublished.Add(uint64(count))
	o.otelMetrics.AddPublishedBatch(ctx, int64(count)) //nolint:gosec // disable G115
}

func (o *metricsRecorder) AddReceivedEvents(ctx context.Context, count uint) {
	o.inputMetrics.eventsReceived.Add(uint64(count))
	o.otelMetrics.AddReceivedEvents(ctx, int64(count)) //nolint:gosec // disable G115
}

func (o *metricsRecorder) AddPublishedEvents(ctx context.Context, count uint) {
	o.inputMetrics.eventsPublished.Add(uint64(count))
	o.otelMetrics.AddPublishedEvents(ctx, int64(count)) //nolint:gosec // disable G115
}

func (o *metricsRecorder) AddProgramExecution(ctx context.Context) {
	o.inputMetrics.executions.Add(1)
	o.otelMetrics.AddProgramExecutionStarted(ctx, 1)
}

func (o *metricsRecorder) AddProgramSuccessExecution(ctx context.Context) {
	o.otelMetrics.AddProgramExecutionSuccess(ctx, 1)
}

func (o *metricsRecorder) SetResourceURL(url string) {
	o.inputMetrics.resource.Set(url)
}
