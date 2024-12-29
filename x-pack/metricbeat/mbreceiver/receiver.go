// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type metricbeatReceiver struct {
	beat   *beat.Beat
	beater beat.Beater
	logger *zap.Logger
}

func (mb *metricbeatReceiver) Start(ctx context.Context, host component.Host) error {
	go func() {
		mb.logger.Info("starting metricbeat receiver")
		err := mb.beater.Run(mb.beat)
		if err != nil {
			mb.logger.Error("metricbeat receiver run error", zap.Error(err))
		}
	}()
	return nil
}

func (mb *metricbeatReceiver) Shutdown(ctx context.Context) error {
	mb.logger.Info("stopping metricbeat receiver")
	mb.beater.Stop()
	return nil
}
