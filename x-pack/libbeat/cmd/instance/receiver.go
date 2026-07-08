// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/management/status"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
)

// contextStopper is an optional extension of beat.Beater for beaters that can
// respect a context deadline while waiting to reach their ready state. If a
// beater implements this, Shutdown uses it so the OTel context deadline is not
// consumed by the ready-state wait before Disconnect is called.
type contextStopper interface {
	StopWithContext(ctx context.Context)
}

// BaseReceiver holds common configurations for beatreceivers.
type BeatReceiver struct {
<<<<<<< HEAD
	beat   *instance.Beat
	beater beat.Beater
	Logger *logp.Logger
=======
	beat                *instance.Beat
	beater              beat.Beater
	reporter            *log.Reporter
	Logger              *logp.Logger
	bridge              *oteltelemetry.RegistryBridge
	releaseSystemBridge func()
	runDone             chan error // receives the error from beater.Run; closed when Run returns
>>>>>>> af686c255 (synchronize filebeat run and shutdown functions (#51800))
}

// NewBeatReceiver creates a BeatReceiver.  This will also create the beater and start the monitoring server if configured
func NewBeatReceiver(ctx context.Context, b *instance.Beat, creator beat.Creator) (BeatReceiver, error) {
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
		retryer := backoff.NewRetryer(50, 100*time.Millisecond, 1*time.Second)
		err := retryer.Retry(ctx, func() error {
			var err error
			b.API, err = api.NewWithDefaultRoutes(
				logp.L().Named("metrics.http"),
				b.Config.HTTP,
				b.Monitoring.InfoRegistry(),
				b.Monitoring.StateRegistry(),
				b.Monitoring.StatsRegistry(),
				b.Monitoring.InputsRegistry())
			if err != nil {
				return fmt.Errorf("could not start the HTTP server for the API: %w", err)
			}
			b.API.Start()
			return nil
		})
		if err != nil {
			return BeatReceiver{}, fmt.Errorf("error creating api listener after 100 retries: %w", err)
		}
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
	var groupReporter otelstatus.RunnerReporter
	if w, ok := br.beater.(cfgfile.WithOtelFactoryWrapper); ok {
		groupReporter = otelstatus.NewGroupStatusReporter(host)
		w.WithOtelFactoryWrapper(otelstatus.StatusReporterFactory(groupReporter))
	}

	// We go through all extensions to find any that implement the DiagnosticExtension interface.
	// This is done so that we can register a diagnostic hook to collect beat metrics.
	extensions := host.GetExtensions()
	for _, ext := range extensions {
		if diagExt, ok := ext.(otelmanager.DiagnosticExtension); ok {
			// if the manager also implements WithDiagnosticExtension interface then set the extension.
			if m, ok := br.beat.Manager.(otelmanager.WithDiagnosticExtension); ok {
				m.SetDiagnosticExtension(br.beat.Info.ComponentID, diagExt)
			}

			// Register a diagnostic hook to collect beat metrics.
			// This is registered once per beat receiver.
			diagExt.RegisterDiagnosticHook(br.beat.Info.ComponentID, "Metrics from the default monitoring namespace and expvar.",
				"beat_metrics.json", "application/json", func() []byte {
					m := monitoring.CollectStructSnapshot((br.beat.Monitoring.StatsRegistry()), monitoring.Full, true)
					data, err := json.MarshalIndent(m, "", "  ")
					if err != nil {
						return fmt.Appendf(nil, "Failed to collect beat metric snapshot for Agent diagnostics: %v", err)
					}
					return data
				})
		}
	}

	br.beat.Manager.SetStopCallback(func() {
		if c, ok := br.beat.Publisher.(io.Closer); ok {
			if err := c.Close(); err != nil {
				br.Logger.Errorf("error closing beat receiver publisher: %v", err)
			}
		}

	})

	br.runDone = make(chan error, 1)
	go func() {
		err := br.beater.Run(&br.beat.Beat)
		if err != nil {
			groupReporter.UpdateStatus(status.Failed, err.Error())
		}
		br.runDone <- err
	}()
	return nil
}

<<<<<<< HEAD
// BeatReceiver.Stop() stops beat receiver.
func (br *BeatReceiver) Shutdown() error {
	br.beater.Stop()

=======
// BeatReceiver.Shutdown stops the beat receiver. The supplied context bounds
// how long the publisher pipeline waits for outstanding acknowledgments before
// it is force-closed (issue #49794); if it carries no deadline the pipeline's
// configured close timeout is used.
func (br *BeatReceiver) Shutdown(ctx context.Context) error {
	if br.bridge != nil {
		br.bridge.Shutdown()
	}
	if br.releaseSystemBridge != nil {
		br.releaseSystemBridge()
	}
	// The Beater owns shutdown sequencing: stop it first so it can close its
	// inputs and finalize acknowledgments before the pipeline is disconnected.
	// See https://github.com/elastic/beats/issues/49794.
	// Pass ctx so beaters that implement contextStopper don't exhaust the OTel
	// shutdown deadline during their ready-state wait.
	if cs, ok := br.beater.(contextStopper); ok {
		cs.StopWithContext(ctx)
	} else {
		br.beater.Stop()
	}

	// Trigger the stop callback. Some beaters (e.g. metricbeat) call
	// Manager.Stop() in their Run() method, but others (e.g. packetbeat in
	// static mode) do not. The OtelManager.stopOnce ensures the callback runs
	// exactly once regardless.
	br.beat.Manager.Stop()

	// Wait for beater.Run to return before disconnecting the pipeline, so the
	// beater owns its full shutdown drain and the pipeline is not torn down
	// while inputs are still active. Bounded by ctx so a hung beater does not
	// block Shutdown indefinitely.
	select {
	case <-br.runDone:
	case <-ctx.Done():
	}

	// Now disconnect the publisher pipeline (this waits for outstanding events
	// to be acknowledged, bounded by the caller's context deadline or the
	// pipeline's configured close timeout). For a receiver sharing an intake
	// queue this disconnects only this pipeline and waits for its own events,
	// leaving co-tenant receivers untouched.
	if err := br.beat.Publisher.Disconnect(ctx); err != nil {
		br.Logger.Errorf("error closing beat receiver publisher: %v", err)
	}

>>>>>>> af686c255 (synchronize filebeat run and shutdown functions (#51800))
	br.beat.Instrumentation.Tracer().Close()
	proc := br.beat.GetProcessors()
	if err := proc.Close(); err != nil {
		br.beat.Info.Logger.Warnf("failed to close global processing: %s", err)
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
