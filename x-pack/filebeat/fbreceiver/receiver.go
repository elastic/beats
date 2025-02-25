// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

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
	"github.com/elastic/elastic-agent-libs/monitoring"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type filebeatReceiver struct {
	beat     *instance.Beat
	beater   beat.Beater
	logger   *zap.Logger
	wg       sync.WaitGroup
	httpConf *config.C
}

func (fb *filebeatReceiver) Start(ctx context.Context, host component.Host) error {
	fb.wg.Add(1)
	go func() {
		defer fb.wg.Done()
		if err := fb.startMonitoring(); err != nil {
			fb.logger.Error("could not start the HTTP server for the API", zap.Error(err))
		}
		if err := fb.beater.Run(&fb.beat.Beat); err != nil {
			fb.logger.Error("filebeat receiver run error", zap.Error(err))
		}
	}()
	return nil
}

func (fb *filebeatReceiver) Shutdown(ctx context.Context) error {
	fb.logger.Info("stopping filebeat receiver")
	fb.beater.Stop()
	if err := fb.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	fb.wg.Wait()
	return nil
}

func (fb *filebeatReceiver) startMonitoring() error {
	if fb.httpConf.Enabled() {
		var err error
		fb.beat.RegisterMetrics()
		err = metricreport.SetupMetrics(logp.NewLogger("metrics"), fb.beat.Info.Beat, version.GetDefaultVersion())
		if err != nil {
			return err
		}
		fb.beat.API, err = api.NewWithDefaultRoutes(logp.NewLogger(""), fb.httpConf, monitoring.GetNamespace)
		if err != nil {
			return err
		}
		fb.beat.API.Start()
	}
	return nil
}

func (fb *filebeatReceiver) stopMonitoring() error {
	if fb.beat.API != nil {
		return fb.beat.API.Stop()
	}
	return nil
}
