// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OTELCELMetrics struct {
	log                                  *logp.Logger
	manualExportFunc                     func(context.Context) error
	exportLock                           sync.Mutex
	started                              bool
	periodicRunCount                     metric.Int64Counter
	periodicBatchGeneratedCount          metric.Int64Counter
	periodicBatchPublishedCount          metric.Int64Counter
	periodicEventGeneratedCount          metric.Int64Counter
	periodicEventPublishedCount          metric.Int64Counter
	periodicRunDuration                  metric.Float64Counter
	periodicCelDuration                  metric.Float64Counter
	periodicEventPublishDuration         metric.Float64Counter
	periodicProgramRunStartedCount       metric.Int64Counter
	periodicProgramRunSuccessCount       metric.Int64Counter
	programBatchProcessedHistogram       metric.Int64Histogram
	programBatchPublishedHistogram       metric.Int64Histogram
	programEventGeneratedHistogram       metric.Int64Histogram
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
func (o *OTELCELMetrics) StartPeriodic() {
	o.exportLock.Lock() // Acquire the lock
	defer o.exportLock.Unlock()
	o.started = true
}

// EndPeriodic ends the periodic metrics collection and manually exports metrics if a manual export function is set.
func (o *OTELCELMetrics) EndPeriodic(ctx context.Context) {
	o.exportLock.Lock() // Acquire the lock
	defer o.exportLock.Unlock()
	if o.started {
		o.log.Debug("OTELCELMetrics EndPeriodic called")
		o.started = false
		if o.manualExportFunc != nil {
			o.log.Debug("OTELCELMetrics manual export started")
			err := o.manualExportFunc(ctx)
			if err != nil {
				o.log.Errorf("error exporting metrics: %v", err)
			}
			o.log.Debug("OTELCELMetrics manual export ended")
		}
	}
}

func (o *OTELCELMetrics) AddPeriodicRun(ctx context.Context, count int64) {
	o.periodicRunCount.Add(ctx, count)
}

func (o *OTELCELMetrics) AddTotalDuration(ctx context.Context, duration time.Duration) {
	o.periodicRunDuration.Add(ctx, duration.Seconds())
	o.programRunDurationHistogram.Record(ctx, duration.Seconds())
}

func (o *OTELCELMetrics) AddPublishDuration(ctx context.Context, duration time.Duration) {
	o.periodicEventPublishDuration.Add(ctx, duration.Seconds())
	o.programEventPublishDurationHistogram.Record(ctx, duration.Seconds())
}
func (o *OTELCELMetrics) AddCELDuration(ctx context.Context, duration time.Duration) {
	o.periodicCelDuration.Add(ctx, duration.Seconds())
	o.programCelDurationHistogram.Record(ctx, duration.Seconds())
}
func (o *OTELCELMetrics) AddGeneratedBatch(ctx context.Context, count int64) {
	o.periodicBatchGeneratedCount.Add(ctx, count)
	o.programBatchProcessedHistogram.Record(ctx, count)
}
func (o *OTELCELMetrics) AddPublishedBatch(ctx context.Context, count int64) {
	o.periodicBatchPublishedCount.Add(ctx, count)
	o.programBatchPublishedHistogram.Record(ctx, count)
}
func (o *OTELCELMetrics) AddEvents(ctx context.Context, count int64) {
	o.periodicEventGeneratedCount.Add(ctx, count)
	o.programEventGeneratedHistogram.Record(ctx, count)
}
func (o *OTELCELMetrics) AddPublishedEvents(ctx context.Context, count int64) {
	o.periodicEventPublishedCount.Add(ctx, count)
	o.programEventPublishedHistogram.Record(ctx, count)
}

func (o *OTELCELMetrics) AddProgramExecution(ctx context.Context, count int64) {
	o.periodicProgramRunStartedCount.Add(ctx, count)
}

func (o *OTELCELMetrics) AddProgramSuccessExecution(ctx context.Context, count int64) {
	o.periodicProgramRunSuccessCount.Add(ctx, count)
}

// Shutdown(ctx context.Context) error
// Attempts to write our metrics. May fail if the contect is cancelled
func (o *OTELCELMetrics) Shutdown(ctx context.Context) error {
	o.EndPeriodic(ctx)
	var err error
	return err
}

// NewOTELCELMetrics initializes a new instance of OTELCELMetrics.
//
// Parameters:
//   - log: A logger instance for logging debug and error messages.
//   - input: A string representing the input source or identifier. Usually the datastream name.
//   - resource: The OpenTelemetry resource containing metadata about the metrics exporter.
//   - tripper: An HTTP RoundTripper to be wrapped by otelhttp.NewTransport.
//   - metricExporter: The OpenTelemetry Metric Exporter that will handle exporting metrics to an endpoint.
//
// Returns:
//   - *OTELCELMetrics: A pointer to a new OTELCELMetrics instance, wrapped in an otelhttp.Transport, and any error encountered during initialization.
func NewOTELCELMetrics(log *logp.Logger,
	input string,
	resource resource.Resource,
	tripper http.RoundTripper,
	metricExporter sdkmetric.Exporter) (*OTELCELMetrics, *otelhttp.Transport, error) {
	var manualExportFunc func(context.Context) error
	var meterProvider metric.MeterProvider

	if metricExporter == nil {
		meterProvider = noop.NewMeterProvider()
	} else {
		reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(DeltaSelector))

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
		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(&resource),
			sdkmetric.WithView(exponentialView))

		manualExportFunc = func(ctx context.Context) error {
			collectedMetrics := &metricdata.ResourceMetrics{}
			err := reader.Collect(ctx, collectedMetrics)
			if err != nil {
				return err
			}
			if log.IsDebug() {
				jsonData, err := json.Marshal(collectedMetrics)
				if err == nil {
					log.Debugf("OTELCELMetrics Collected metrics %s", jsonData)
				} else {
					log.Debugf("OTELCELMetrics could not marshall Collected metrics into json %v", collectedMetrics)
				}
			}
			go func(ctx context.Context, log *logp.Logger, metricExporter sdkmetric.Exporter, collectedMetrics *metricdata.ResourceMetrics) {
				err := metricExporter.Export(ctx, collectedMetrics)
				if err != nil {
					log.Error("Failed to export metrics: ", err)
				}
			}(ctx, log, metricExporter, collectedMetrics)
			return nil
		}
	}
	transport := otelhttp.NewTransport(tripper, otelhttp.WithMeterProvider(meterProvider))

	meter := meterProvider.Meter("github.com/elastic/beats/x-pack/filebeat/otel/cel_metrics.go")

	periodicRunCount, err := meter.Int64Counter("input.cel.periodic.run")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.run: %w", err)
	}
	programRunStartedCount, err := meter.Int64Counter("input.cel.periodic.program.run.started")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.started: %w", err)
	}
	programRunSuccessCount, err := meter.Int64Counter("input.cel.periodic.program.run.success")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.success: %w", err)
	}
	periodicBatchCount, err := meter.Int64Counter("input.cel.periodic.batch.generated")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.batch.generated: %w", err)
	}
	periodicPublishedBatchCount, err := meter.Int64Counter("input.cel.periodic.batch.published")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.batch.published: %w", err)
	}
	periodicEventCount, err := meter.Int64Counter("input.cel.periodic.event.generated")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event: %w", err)
	}
	periodicPublishedEventCount, err := meter.Int64Counter("input.cel.periodic.event.published")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event.published: %w", err)
	}
	periodicTotalDuration, err := meter.Float64Counter("input.cel.periodic.run.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.run.duration: %w", err)
	}
	periodicCELDuration, err := meter.Float64Counter("input.cel.periodic.cel.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.cel.duration: %w", err)
	}
	periodicPublishDuration, err := meter.Float64Counter("input.cel.periodic.event.publish.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.event.publish.duration: %w", err)
	}

	programBatchProcessed, err := meter.Int64Histogram("input.cel.program.batch.processed")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.processed: %w", err)
	}
	programBatchPublished, err := meter.Int64Histogram("input.cel.program.batch.published")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.published: %w", err)
	}
	programEventGenerated, err := meter.Int64Histogram("input.cel.program.event.generated")
	if err != nil {
		return nil, nil, fmt.Errorf("failed"+
			" to create input.cel.program.event.generated: %w", err)
	}
	programEventPublished, err := meter.Int64Histogram("input.cel.program.event.published")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.event.published: %w", err)
	}

	programRunDuration, err := meter.Float64Histogram("input.cel.program.run.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.duration: %w", err)
	}
	programCELDuration, err := meter.Float64Histogram("input.cel.program.cel.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.cel.duration: %w", err)
	}
	programPublishDuration, err := meter.Float64Histogram("input.cel.program.publish.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.publish.duration: %w", err)
	}

	return &OTELCELMetrics{
		log:                                  log,
		manualExportFunc:                     manualExportFunc,
		periodicRunCount:                     periodicRunCount,
		periodicBatchGeneratedCount:          periodicBatchCount,
		periodicBatchPublishedCount:          periodicPublishedBatchCount,
		periodicEventGeneratedCount:          periodicEventCount,
		periodicEventPublishedCount:          periodicPublishedEventCount,
		periodicRunDuration:                  periodicTotalDuration,
		periodicCelDuration:                  periodicCELDuration,
		periodicEventPublishDuration:         periodicPublishDuration,
		periodicProgramRunStartedCount:       programRunStartedCount,
		periodicProgramRunSuccessCount:       programRunSuccessCount,
		programBatchProcessedHistogram:       programBatchProcessed,
		programBatchPublishedHistogram:       programBatchPublished,
		programEventGeneratedHistogram:       programEventGenerated,
		programEventPublishedHistogram:       programEventPublished,
		programRunDurationHistogram:          programRunDuration,
		programCelDurationHistogram:          programCELDuration,
		programEventPublishDurationHistogram: programPublishDuration,
	}, transport, nil

}
