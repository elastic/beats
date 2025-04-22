// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type metricbeatReceiver struct {
	beat     *instance.Beat
	beater   beat.Beater
	logger   *zap.Logger
	wg       sync.WaitGroup
	httpConf *config.C
}

func (mb *metricbeatReceiver) Start(ctx context.Context, host component.Host) error {
	mb.wg.Add(1)
	go func() {
		defer mb.wg.Done()
		mb.logger.Info("starting metricbeat receiver")
		if err := mb.startMonitoring(); err != nil {
			mb.logger.Error("could not start the HTTP server for the monitoring API", zap.Error(err))
		}
		if err := mb.beater.Run(&mb.beat.Beat); err != nil {
			mb.logger.Error("metricbeat receiver run error", zap.Error(err))
		}
	}()
	return nil
}

func (mb *metricbeatReceiver) Shutdown(ctx context.Context) error {
	mb.logger.Info("stopping metricbeat receiver")
	mb.beater.Stop()
	if err := mb.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	mb.wg.Wait()
	return nil
}

func (mb *metricbeatReceiver) startMonitoring() error {
	if !mb.httpConf.Enabled() {
		return nil
	}
	var err error

	mb.beat.RegisterMetrics()

	statsReg := mb.beat.Info.Monitoring.StatsRegistry

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

	err = metricreport.SetupMetrics(logp.NewLogger("metrics"), mb.beat.Info.Beat, version.GetDefaultVersion(), metricreport.WithProcessRegistry(processReg), metricreport.WithSystemRegistry(systemReg))
	if err != nil {
		return err
	}
	mb.beat.API, err = api.NewWithDefaultRoutes(logp.NewLogger("metrics.http"), mb.httpConf, api.RegistryLookupFunc(mb.beat.Info.Monitoring.Namespace))
	if err != nil {
		return err
	}
	mb.beat.API.Start()

	return nil
}

func (mb *metricbeatReceiver) stopMonitoring() error {
	if mb.beat.API == nil {
		return nil
	}
	return mb.beat.API.Stop()
}
