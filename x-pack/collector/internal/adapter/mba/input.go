package mba

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
	"github.com/urso/sderr"
)

type metricsetInput struct {
	inputName     string
	moduleName    string
	metricsetName string
	namespace     string
	tasks         []mb.MetricSet
	modifiers     []mb.EventModifier
}

func (m *metricsetInput) Name() string { return m.inputName }

func (m *metricsetInput) Test(_ v2.TestContext) error {
	// metricsets do not support active testing
	return nil
}

func (m *metricsetInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	var grp unison.MultiErrGroup
	for _, task := range m.tasks {
		task := task
		grp.Go(func() error {
			inpCtx := ctx
			inpCtx.ID = task.ID()
			inpCtx.Logger = ctx.Logger.With("task", task.ID())

			return m.runTask(inpCtx, task, pipeline)
		})
	}

	if errs := grp.Wait(); len(errs) > 0 {
		return sderr.WrapAll(errs, "input %{id} failed", ctx.ID)
	}
	return nil
}

func (m *metricsetInput) runTask(ctx v2.Context, ms mb.MetricSet, pipeline beat.Pipeline) error {
	client, err := pipeline.Connect()
	if err != nil {
		return err
	}

	// setup internal metrics collection
	metricsPath := ms.ID()
	stats := getMetricSetStats(m.inputName)
	defer releaseStats(stats)
	registry := monitoring.GetNamespace("dataset").GetRegistry()
	defer registry.Remove(metricsPath)
	registry.Add(metricsPath, ms.Metrics(), monitoring.Full)
	monitoring.NewString(ms.Metrics(), "starttime").Set(common.Time(time.Now()).String())

	// event event finalization and publishing
	reporter := &eventReporter{
		cancel: ctx.Cancelation,
		client: client,
		stats:  stats,
		eventTransformer: eventTransformer{
			inputName: m.inputName,
			namespace: m.namespace,
			metricset: ms,
			modifiers: m.modifiers,
		},
	}

	// run the metricset fetch and publish loop
	switch ms := ms.(type) {
	case mb.PushMetricSet:
		ms.Run(reporter.V1())
	case mb.PushMetricSetV2:
		ms.Run(reporter.V2())
	case mb.PushMetricSetV2WithContext:
		ms.Run(ctxtool.FromCanceller(ctx.Cancelation), reporter.V2())
	case mb.ReportingMetricSet, mb.ReportingMetricSetV2, mb.ReportingMetricSetV2Error, mb.ReportingMetricSetV2WithContext:
		{
			reporter.eventTransformer.periodic = true
			stopCtx := ctxtool.FromCanceller(ctx.Cancelation)
			reporter.StartFetchTimer()
			m.fetchAndReport(stopCtx, ms, reporter)
			timed.Periodic(stopCtx, ms.Module().Config().Period, func() error {
				reporter.StartFetchTimer()
				m.fetchAndReport(stopCtx, ms, reporter)
				return nil
			})
		}
	default:
		// Earlier startup stages prevent this from happening.
		logp.Err("MetricSet '%s' does not implement an event producing interface", m.inputName)
	}
	return nil
}

func (m *metricsetInput) fetchAndReport(ctx context.Context, ms mb.MetricSet, reporter reporter) {
	switch fetcher := ms.(type) {
	case mb.ReportingMetricSet:
		fetcher.Fetch(reporter.V1())
	case mb.ReportingMetricSetV2:
		fetcher.Fetch(reporter.V2())
	case mb.ReportingMetricSetV2Error:
		err := fetcher.Fetch(reporter.V2())
		if err != nil {
			reporter.V2().Error(err)
			logp.Info("Error fetching data for metricset %s: %s", m.inputName, err)
		}
	case mb.ReportingMetricSetV2WithContext:
		err := fetcher.Fetch(ctx, reporter.V2())
		if err != nil {
			reporter.V2().Error(err)
			logp.Info("Error fetching data for metricset %s: %s", m.inputName, err)
		}
	default:
		panic(fmt.Sprintf("unexpected fetcher type for %v", ms))
	}
}
