// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/status"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/elastic-agent-libs/logp"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
)

// BaseReceiver holds common configurations for beatreceivers.
type BeatReceiver struct {
	beat   *instance.Beat
	beater beat.Beater
	Logger *logp.Logger
}

// NewBeatReceiver creates a BeatReceiver.  This will also create the beater and start the monitoring server if configured
func NewBeatReceiver(b *instance.Beat, creator beat.Creator) (BeatReceiver, error) {
	beatConfig, err := b.BeatConfig()
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error getting beat config: %w", err)
	}

	b.RegisterMetrics()

	statsReg := b.Monitoring.StatsRegistry()

	// stats.beat
	processReg := statsReg.GetOrCreateRegistry("beat")

	// stats.system
	systemReg := statsReg.GetOrCreateRegistry("system")

	err = metricreport.SetupMetricsOptions(metricreport.MetricOptions{
		Logger:         b.Info.Logger.Named("metrics"),
		Name:           b.Info.Name,
		Version:        b.Info.Version,
		SystemMetrics:  systemReg,
		ProcessMetrics: processReg,
	})
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error setting up metrics report: %w", err)
	}

	if b.Config.HTTP.Enabled() {
		var err error
		b.API, err = api.NewWithDefaultRoutes(
			b.Info.Logger.Named("metrics.http"),
			b.Config.HTTP,
			b.Monitoring.InfoRegistry(),
			b.Monitoring.StateRegistry(),
			b.Monitoring.StatsRegistry(),
			b.Monitoring.InputsRegistry())
		if err != nil {
			return BeatReceiver{}, fmt.Errorf("could not start the HTTP server for the API: %w", err)
		}
		b.API.Start()
	}

	beater, err := creator(&b.Beat, beatConfig)
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error getting %s creator:%w", b.Info.Beat, err)
	}
	return BeatReceiver{
		beat:   b,
		beater: beater,
		Logger: b.Info.Logger,
	}, nil
}

// BeatReceiver.Start() starts the beat receiver.
func (br *BeatReceiver) Start(host component.Host) error {
	if w, ok := br.beater.(cfgfile.WithOtelFactoryWrapper); ok {
		groupReporter := status.NewGroupStatusReporter(host)
		w.WithOtelFactoryWrapper(status.StatusReporterFactory(groupReporter))
	}

	if err := br.beater.Run(&br.beat.Beat); err != nil {
		return fmt.Errorf("beat receiver run error: %w", err)
	}

	return nil
}

// BeatReceiver.Stop() stops beat receiver.
func (br *BeatReceiver) Shutdown() error {
	br.beater.Stop()

	br.beat.Instrumentation.Tracer().Close()
	proc := br.beat.GetProcessors()
	if err := proc.Close(); err != nil {
		br.beat.Info.Logger.Warnf("failed to close global processing: %s", err)
	}

	if c, ok := br.beat.Publisher.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return fmt.Errorf("error closing beat receiver publisher: %w", err)
		}
	}

	if err := br.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	if err := br.beat.Info.Logger.Close(); err != nil {
		return fmt.Errorf("error closing beat receiver logging: %w", err)
	}
	return nil
}

func (br *BeatReceiver) stopMonitoring() error {
	if br.beat.API != nil {
		return br.beat.API.Stop()
	}
	return nil
}
