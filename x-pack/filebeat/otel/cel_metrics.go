package otel

import (
	"context"
	"errors"
	"fmt"
	"github.com/elastic/elastic-agent-libs/logp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"net/http"
	"time"
)

type OTELCELMetrics struct {
	log                          *logp.Logger
	shutdownFuncs                []func(context.Context) error
	flushFuncs                   []func(context.Context) error
	manualExportFunc             func(context.Context) error
	started                      bool
	periodicRunCount             metric.Int64Counter
	periodicProgramStarted       metric.Int64Counter
	periodicProgramSuccess       metric.Int64Counter
	periodicBatchGenerated       metric.Int64Counter
	periodicBatchPublished       metric.Int64Counter
	periodicEventGenerated       metric.Int64Counter
	periodicEventPublished       metric.Int64Counter
	periodicRunDuration          metric.Float64Counter
	periodicCelDuration          metric.Float64Counter
	periodicEventPublishDuration metric.Float64Counter
	programRunStartedCount       metric.Int64Counter
	programRunSuccessCount       metric.Int64Counter
	programBatchCount            metric.Int64Counter
	programBatchPublishedCount   metric.Int64Counter
	programEventCount            metric.Int64Counter
	programEventPublishedCount   metric.Int64Counter
	programRunDuration           metric.Float64Counter
	programCelDuration           metric.Float64Counter
	programEventPublishDuration  metric.Float64Counter
}

func (o *OTELCELMetrics) StartPeriodic() {
	o.started = true
}

func (o *OTELCELMetrics) EndPeriodic(ctx context.Context) {
	if o.started {
		o.log.Debug("OTELCELMetrics EndPeriodic called")
		o.started = false
		if o.manualExportFunc != nil {
			o.log.Debug("OTELCELMetrics manual export started")
			o.manualExportFunc(ctx)
			o.log.Debug("OTELCELMetrics manual export ended")
		}
	}
}

// push last metrics out to exporter which will then push them to the endpoint.
func (o *OTELCELMetrics) ForceFlush(ctx context.Context, force bool) error {
	o.log.Debug("OTELCELMetrics forcing flush")
	if o.started && !force {
		return errors.New("OTELCELMetrics cannot flush in the middle of a periodic run. Use force == true to force a flush.")
	}
	var err error
	for _, fn := range o.flushFuncs {
		err = errors.Join(err, fn(ctx))
	}
	return err
}

func (o *OTELCELMetrics) AddPeriodicRun(ctx context.Context, count int64) {
	o.periodicRunCount.Add(ctx, count)
}
func (o *OTELCELMetrics) AddTotalDuration(ctx context.Context, duration time.Duration) {
	o.periodicRunDuration.Add(ctx, duration.Seconds())
	o.programRunDuration.Add(ctx, duration.Seconds())
}
func (o *OTELCELMetrics) AddPublishDuration(ctx context.Context, duration time.Duration) {
	o.periodicEventPublishDuration.Add(ctx, duration.Seconds())
	o.programEventPublishDuration.Add(ctx, duration.Seconds())
}
func (o *OTELCELMetrics) AddCELDuration(ctx context.Context, duration time.Duration) {
	o.periodicCelDuration.Add(ctx, duration.Seconds())
	o.programCelDuration.Add(ctx, duration.Seconds())
}
func (o *OTELCELMetrics) AddBatch(ctx context.Context, count int64) {
	o.periodicBatchGenerated.Add(ctx, count)
	o.programBatchCount.Add(ctx, count)
}
func (o *OTELCELMetrics) AddPublishedBatch(ctx context.Context, count int64) {
	o.periodicBatchPublished.Add(ctx, count)
	o.programBatchPublishedCount.Add(ctx, count)
}
func (o *OTELCELMetrics) AddEvents(ctx context.Context, count int64) {
	o.periodicEventGenerated.Add(ctx, count)
	o.programEventCount.Add(ctx, count)
}
func (o *OTELCELMetrics) AddPublishedEvents(ctx context.Context, count int64) {
	o.periodicEventPublished.Add(ctx, count)
	o.programEventPublishedCount.Add(ctx, count)
}

func (o *OTELCELMetrics) AddProgramExecution(ctx context.Context, count int64) {
	o.programRunStartedCount.Add(ctx, count)
}

func (o *OTELCELMetrics) AddProgramSuccessExecution(ctx context.Context, count int64) {
	o.programRunSuccessCount.Add(ctx, count)
}

// Shutdown(ctx context.Context) error
// Flushes the meters to the exporters, then shutsdown the exporter
func (o *OTELCELMetrics) Shutdown(ctx context.Context) error {
	o.EndPeriodic(ctx)
	o.ForceFlush(ctx, true)
	var err error
	for _, fn := range o.shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}
	return err
}

func NewOTELCELMetrics(log *logp.Logger,
	input string,
	resource resource.Resource,
	tripper http.RoundTripper,
	metricExporter sdkmetric.Exporter,
	interval time.Duration) (*OTELCELMetrics, *otelhttp.Transport, error) {
	var shutdownFuncs []func(context.Context) error
	var flushFuncs []func(context.Context) error
	var manualExportFunc func(context.Context) error
	var meterProvider metric.MeterProvider

	if metricExporter == nil {
		meterProvider = noop.NewMeterProvider()
	} else {
		var sdkmeterProvider *sdkmetric.MeterProvider
		if interval == 0 {
			log.Debug("OTELCELMetrics NewMeterProvider called without interval. Creating Manual Export")
			reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(DeltaSelector))
			sdkmeterProvider = sdkmetric.NewMeterProvider(
				sdkmetric.WithReader(reader), sdkmetric.WithResource(&resource))
			manualExportFunc = func(ctx context.Context) error {
				collectedMetrics := &metricdata.ResourceMetrics{}
				err := reader.Collect(ctx, collectedMetrics)
				if err != nil {
					return err
				}
				collectedMetrics = GetHttpCountsFromHistogram(input, collectedMetrics)
				return metricExporter.Export(ctx, collectedMetrics)
			}
			flushFuncs = append(flushFuncs, manualExportFunc)
		} else {
			log.Debug("OTELCELMetrics NewMeterProvider has fixed interval. Creating Periodic Export")
			sdkmeterProvider = sdkmetric.NewMeterProvider(
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
					sdkmetric.WithInterval(interval))), sdkmetric.WithResource(&resource))
			shutdownFuncs = append(shutdownFuncs, sdkmeterProvider.Shutdown)
			flushFuncs = append(flushFuncs, sdkmeterProvider.ForceFlush)
		}
		meterProvider = sdkmeterProvider
	}
	meter := meterProvider.Meter(input)
	transport := otelhttp.NewTransport(tripper, otelhttp.WithMeterProvider(meterProvider))

	periodicTotalRunCount, err := meter.Int64Counter("input.cel.periodic.run.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.run.count: %w", err)
	}
	periodicProgramStarted, err := meter.Int64Counter("input.cel.periodic.program.started")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.program.started: %w", err)
	}
	periodicProgramSuccess, err := meter.Int64Counter("input.cel.periodic.program.success")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.periodic.program.success: %w", err)
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

	programRunStartedCount, err := meter.Int64Counter("input.cel.program.run.started.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.started.count: %w", err)
	}
	programRunSuccessCount, err := meter.Int64Counter("input.cel.program.run.success.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.success.count: %w", err)
	}

	programBatchCount, err := meter.Int64Counter("input.cel.program.batch.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.count: %w", err)
	}

	programEventCount, err := meter.Int64Counter("input.cel.program.event.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.event.count: %w", err)
	}
	programEventPublishedCount, err := meter.Int64Counter("input.cel.program.event.published.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.event.published.count: %w", err)
	}
	programBatchPublishedCount, err := meter.Int64Counter("input.cel.program.batch.published.count")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.batch.published.count: %w", err)
	}
	programRunDuration, err := meter.Float64Counter("input.cel.program.run.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.run.duration: %w", err)
	}
	programCELDuration, err := meter.Float64Counter("input.cel.program.cel.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.cel.duration: %w", err)
	}
	programPublishDuration, err := meter.Float64Counter("input.cel.program.publish.duration")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create input.cel.program.publish.duration: %w", err)
	}

	return &OTELCELMetrics{
		log:                          log,
		shutdownFuncs:                shutdownFuncs,
		flushFuncs:                   flushFuncs,
		manualExportFunc:             manualExportFunc,
		periodicRunCount:             periodicTotalRunCount,
		periodicProgramStarted:       periodicProgramStarted,
		periodicProgramSuccess:       periodicProgramSuccess,
		periodicBatchGenerated:       periodicBatchCount,
		periodicBatchPublished:       periodicPublishedBatchCount,
		periodicEventGenerated:       periodicEventCount,
		periodicEventPublished:       periodicPublishedEventCount,
		periodicRunDuration:          periodicTotalDuration,
		periodicCelDuration:          periodicCELDuration,
		periodicEventPublishDuration: periodicPublishDuration,
		programRunStartedCount:       programRunStartedCount,
		programRunSuccessCount:       programRunSuccessCount,
		programBatchCount:            programBatchCount,
		programBatchPublishedCount:   programBatchPublishedCount,
		programEventCount:            programEventCount,
		programEventPublishedCount:   programEventPublishedCount,
		programRunDuration:           programRunDuration,
		programCelDuration:           programCELDuration,
		programEventPublishDuration:  programPublishDuration,
	}, transport, nil

}
