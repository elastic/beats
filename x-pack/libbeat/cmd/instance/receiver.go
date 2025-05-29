// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/version"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"

	"go.uber.org/zap"
)

// BaseReceiver holds common configurations for beatreceivers.
type BeatReceiver struct {
	beat   *instance.Beat
	beater beat.Beater
	Logger *zap.Logger
}

// reporter implements the status.StatusReporter interface and maps beat statuses to collector statuses.
type reporter struct {
	host component.Host
}

func (r *reporter) UpdateStatus(s status.Status, msg string) {
	switch s {
	case status.Starting:
		componentstatus.ReportStatus(r.host, componentstatus.NewEvent(componentstatus.StatusStarting))
	case status.Running:
		componentstatus.ReportStatus(r.host, componentstatus.NewEvent(componentstatus.StatusOK))
	case status.Degraded:
		componentstatus.ReportStatus(r.host, componentstatus.NewRecoverableErrorEvent(errors.New(msg)))
	case status.Failed:
		componentstatus.ReportStatus(r.host, componentstatus.NewPermanentErrorEvent(errors.New(msg)))
	case status.Stopping:
		componentstatus.ReportStatus(r.host, componentstatus.NewEvent(componentstatus.StatusStopped))
	case status.Stopped:
		componentstatus.ReportStatus(r.host, componentstatus.NewEvent(componentstatus.StatusStopped))
	}
}

// NewBeatReceiver creates a BeatReceiver.  This will also create the beater and start the monitoring server if configured
func NewBeatReceiver(b *instance.Beat, creator beat.Creator, logger *zap.Logger) (BeatReceiver, error) {
	beatConfig, err := b.BeatConfig()
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error getting beat config: %w", err)
	}

	b.RegisterMetrics()

	statsReg := b.Info.Monitoring.StatsRegistry

	// stats.beat
	processReg := statsReg.GetRegistry("beat")
	if processReg == nil {
		processReg = statsReg.NewRegistry("beat")
	}

	// stats.system
	systemReg := statsReg.GetRegistry("system")
	if systemReg == nil {
		systemReg = statsReg.NewRegistry("system")
	}

	err = metricreport.SetupMetrics(b.Info.Logger.Named("metrics"), b.Info.Beat, version.GetDefaultVersion(), metricreport.WithProcessRegistry(processReg), metricreport.WithSystemRegistry(systemReg))
	if err != nil {
		return BeatReceiver{}, fmt.Errorf("error setting up metrics report: %w", err)
	}

	if b.Config.HTTP.Enabled() {
		var err error
		b.API, err = api.NewWithDefaultRoutes(b.Info.Logger.Named("metrics.http"), b.Config.HTTP, api.RegistryLookupFunc(b.Info.Monitoring.Namespace))
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
		Logger: logger,
	}, nil
}

// BeatReceiver.Stop() starts the beat receiver.
func (br *BeatReceiver) Start(host component.Host) error {
	br.beat.OtelStatusReporter = &reporter{host: host}
	if err := br.beater.Run(&br.beat.Beat); err != nil {
		return fmt.Errorf("beat receiver run error: %w", err)
	}
	return nil
}

// BeatReceiver.Stop() stops beat receiver.
func (br *BeatReceiver) Shutdown() error {
	br.beater.Stop()
	if err := br.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	return nil
}

func (br *BeatReceiver) stopMonitoring() error {
	if br.beat.API != nil {
		return br.beat.API.Stop()
	}
	return nil
}
