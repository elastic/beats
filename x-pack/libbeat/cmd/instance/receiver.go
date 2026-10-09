// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/monitoring/report/log"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	metricreport "github.com/elastic/beats/v7/pkg/systemmetrics/report"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
	oteltelemetry "github.com/elastic/beats/v7/x-pack/otel/telemetry"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
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
	beat                *instance.Beat
	beater              beat.Beater
	reporter            *log.Reporter
	Logger              *logp.Logger
	bridge              *oteltelemetry.RegistryBridge
	releaseSystemBridge func()
	runDone             chan error // receives the error from beater.Run; closed when Run returns
}

// NewBeatReceiver creates a BeatReceiver.  This will also create the beater and start the monitoring server if configured
func NewBeatReceiver(ctx context.Context, b *instance.Beat, creator beat.Creator, set receiver.Settings) (BeatReceiver, error) {
	receiverID := set.ID
	bridgeTelemetrySettings := set.TelemetrySettings
	// The Beat logger is built from the collector logger core and includes the
	// component.* fields configured for this receiver, they're necessary to
	// route the logs to the correct datastream.
	bridgeTelemetrySettings.Logger = zap.New(b.Info.Logger.Core())
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

	bridge, err := oteltelemetry.NewRegistryBridge(bridgeTelemetrySettings, receiverID.String(), b.Monitoring.StatsRegistry(), b.Monitoring.InputsRegistry())
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error creating registry bridge: %w", err)
	}

	// The system bridge is shared by all receivers. Preserve the original
	// settings so its logs are not permanently tagged with this receiver's
	// component.* fields.
	releaseSystem, err := oteltelemetry.AcquireSystemBridge(set.TelemetrySettings)
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error acquiring system bridge: %w", err)
	}

	return BeatReceiver{
		beat:                b,
		beater:              beater,
		Logger:              b.Info.Logger,
		bridge:              bridge,
		releaseSystemBridge: releaseSystem,
		runDone:             make(chan error, 1),
	}, nil
}

// BeatReceiver.Start() starts the beat receiver.
func (br *BeatReceiver) Start(host component.Host) (retErr error) {
	// If Start returns an error before the beater.Run goroutine is launched,
	// signal runDone so that a concurrent Shutdown does not block forever on it.
	defer func() {
		if retErr != nil {
			br.runDone <- retErr
		}
	}()
	var groupReporter otelstatus.RunnerReporter
	if w, ok := br.beater.(cfgfile.WithOtelFactoryWrapper); ok {
		groupReporter = otelstatus.NewGroupStatusReporter(host)
		w.WithOtelFactoryWrapper(otelstatus.StatusReporterFactory(groupReporter))
	}

	// We go through all extensions to find any that implement the DiagnosticExtension or
	// ActionExtension interfaces.
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

		// This is done so that Fleet actions (e.g. osquery live queries) routed to
		// elastic-agent can reach this beat receiver instance.
		if actionExt, ok := ext.(otelmanager.ActionExtension); ok {
			// if the manager also implements WithActionExtension interface then set the extension.
			if m, ok := br.beat.Manager.(otelmanager.WithActionExtension); ok {
				m.SetActionExtension(br.beat.Info.ComponentID, actionExt)
			}
		}
	}

	if ua := userAgentFromHeaders(headersFromExtensions(context.Background(), extensions, br.Logger)); ua != "" {
		br.Logger.Debugf("using User-Agent from headers_setter extension: %q", ua)
		br.beat.Info.UserAgent = ua
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

	go func() {
		err := br.beater.Run(&br.beat.Beat)
		if err != nil {
			groupReporter.UpdateStatus(status.Failed, err.Error())
		}
		br.runDone <- err
	}()
	return nil
}

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

const (
	headersSetterType    = "headers_setter"
	agentComponentPrefix = "_agent-component/"
)

// headersFromExtensions returns the headers configured on headers_setter extension.
// Elastic Agent adds one per collector, shared by all receivers.
func headersFromExtensions(ctx context.Context, extensions map[component.ID]component.Component, log *logp.Logger) map[string]string {
	var ids []component.ID
	for id := range extensions {
		if id.Type().String() == headersSetterType && strings.HasPrefix(id.Name(), agentComponentPrefix) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	if len(ids) > 1 {
		log.Debugf("found %d headers_setter extensions (%v); using %s", len(ids), ids, ids[0])
	}
	id := ids[0]

	grpcClient, ok := extensions[id].(extensionauth.GRPCClient)
	if !ok {
		log.Warnf("extension %s does not implement extensionauth.GRPCClient; its headers cannot be read", id)
		return nil
	}
	creds, err := grpcClient.PerRPCCredentials()
	if err != nil {
		// Only fails when additional_auth is configured and that extension is
		// missing or broken.
		log.Warnf("extension %s: could not get PerRPCCredentials, ignoring its headers: %v", id, err)
		return nil
	}
	if creds == nil {
		return nil
	}
	headers, err := creds.GetRequestMetadata(ctx)
	if err != nil {
		log.Warnf("extension %s: could not resolve headers, ignoring them: %v", id, err)
		return nil
	}
	return headers
}

// userAgentFromHeaders returns the User-Agent value in headers, matched
// case-insensitively, or "" if there is none.
func userAgentFromHeaders(headers map[string]string) string {
	for k, v := range headers {
		if strings.EqualFold(k, "User-Agent") {
			return v
		}
	}
	return ""
}
