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
	"github.com/elastic/beats/v7/libbeat/monitoring/report/log"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
	oteltelemetry "github.com/elastic/beats/v7/x-pack/otel/telemetry"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

// BaseReceiver holds common configurations for beatreceivers.
type BeatReceiver struct {
	beat                *instance.Beat
	beater              beat.Beater
	reporter            *log.Reporter
	Logger              *logp.Logger
	bridge              *oteltelemetry.RegistryBridge
	releaseSystemBridge func()
}

// NewBeatReceiver creates a BeatReceiver.  This will also create the beater and start the monitoring server if configured
func NewBeatReceiver(ctx context.Context, b *instance.Beat, creator beat.Creator, set receiver.Settings) (BeatReceiver, error) {
	receiverID := set.ID
	ts := set.TelemetrySettings
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
				b.Info.Logger.Named("metrics.http"),
				b.Config.HTTP,
				b.Monitoring)
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

	bridge, err := oteltelemetry.NewRegistryBridge(ts, receiverID.String(), b.Monitoring.StatsRegistry(), b.Monitoring.InputsRegistry())
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error creating registry bridge: %w", err)
	}

	releaseSystem, err := oteltelemetry.AcquireSystemBridge(ts)
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error acquiring system bridge: %w", err)
	}

	return BeatReceiver{
		beat:                b,
		beater:              beater,
		Logger:              b.Info.Logger,
		bridge:              bridge,
		releaseSystemBridge: releaseSystem,
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

	if w, ok := br.beater.(backend.WithESStateStoreExtension); ok {
		if present, err := br.beat.RawConfig.Has("storage", -1); present && err == nil {
			storageID, err := br.beat.RawConfig.String("storage", -1)
			if err != nil {
				return fmt.Errorf("error reading storage extension from config: %w", err)
			}
			esStorageExtension, err := br.getESStateStoreExtension(host, storageID)
			if err != nil {
				return fmt.Errorf("error getting ES state store extension: %w", err)
			}
			w.WithESStateStoreExtension(esStorageExtension)
		}
	}

	if br.beat.Config.MetricLogging == nil || br.beat.Config.MetricLogging.Enabled() {
		r, err := log.MakeReporter(br.beat.Info,
			br.beat.Config.MetricLogging,
			br.beat.Monitoring)
		if err != nil {
			return fmt.Errorf("error creating metric reporter: %w", err)
		}
		rep, ok := r.(*log.Reporter)
		if !ok {
			return fmt.Errorf("error creating metric log reporter")
		}
		br.reporter = rep
	}

	br.beat.Manager.SetStopCallback(func() {
		if c, ok := br.beat.Publisher.(io.Closer); ok {
			if err := c.Close(); err != nil {
				br.Logger.Errorf("error closing beat receiver publisher: %v", err)
			}
		}
	})

	if err := br.beater.Run(&br.beat.Beat); err != nil {
		// set beatreceiver status
		groupReporter.UpdateStatus(status.Failed, err.Error())
		return fmt.Errorf("beat receiver run error: %w", err)
	}

	return nil
}

// BeatReceiver.Stop() stops beat receiver.
func (br *BeatReceiver) Shutdown() error {
	if br.bridge != nil {
		br.bridge.Shutdown()
	}
	if br.releaseSystemBridge != nil {
		br.releaseSystemBridge()
	}
	br.beater.Stop()

	br.beat.Instrumentation.Tracer().Close()
	proc := br.beat.GetProcessors()
	if err := proc.Close(); err != nil {
		br.beat.Info.Logger.Warnf("failed to close global processing: %s", err)
	}

	if err := br.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}

	if br.reporter != nil {
		br.reporter.Stop()
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

func (br *BeatReceiver) getESStateStoreExtension(host component.Host, storageExtension string) (backend.Registry, error) {
	componentID := component.ID{}
	err := componentID.UnmarshalText([]byte(storageExtension))
	if err != nil {
		return nil, fmt.Errorf("invalid component id for ES state store extension (%v): %w", []byte(storageExtension), err)
	}
	extension, ok := host.GetExtensions()[componentID]
	if !ok {
		return nil, fmt.Errorf("extension with id %s not found", componentID.String())
	}
	reg, ok := extension.(backend.Registry)
	if !ok {
		return nil, fmt.Errorf("extension '%s' is not a backend.Registry", componentID.String())
	}
	return reg, nil
}
