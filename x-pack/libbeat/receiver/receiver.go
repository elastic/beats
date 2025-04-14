// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package receiver

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"
)

// BaseReceiver holds common configurations for fbreceiver and mbreceiver.
type BaseReceiver struct {
	HttpConf *config.C
	Beat     *instance.Beat
	Beater   beat.Beater
}

func (b *BaseReceiver) Start() error {
	if err := b.startMonitoring(); err != nil {
		return fmt.Errorf("could not start the HTTP server for the monitoring API: %w", err)
	}
	return nil
}

func (b *BaseReceiver) Shutdown() error {
	if err := b.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	return nil
}

func (b *BaseReceiver) startMonitoring() error {
	if b.HttpConf.Enabled() {
		var err error

		b.Beat.RegisterMetrics()

		statsReg := b.Beat.Info.Monitoring.StatsRegistry

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

		err = metricreport.SetupMetrics(logp.NewLogger("metrics"), b.Beat.Info.Beat, version.GetDefaultVersion(), metricreport.WithProcessRegistry(processReg), metricreport.WithSystemRegistry(systemReg))
		if err != nil {
			return err
		}
		b.Beat.API, err = api.NewWithDefaultRoutes(logp.NewLogger("metrics.http"), b.HttpConf, api.RegistryLookupFunc(b.Beat.Info.Monitoring.Namespace))
		if err != nil {
			return err
		}
		b.Beat.API.Start()
	}
	return nil
}

func (b *BaseReceiver) stopMonitoring() error {
	if b.Beat.API != nil {
		return b.Beat.API.Stop()
	}
	return nil
}
