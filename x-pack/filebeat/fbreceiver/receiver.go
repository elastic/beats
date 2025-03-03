// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type filebeatReceiver struct {
	beat   *beat.Beat
	beater beat.Beater
	logger *zap.Logger
	wg     sync.WaitGroup
}

func (fb *filebeatReceiver) Start(ctx context.Context, host component.Host) error {
	fb.wg.Add(1)
	go func() {
		defer fb.wg.Done()
		fb.logger.Info("starting filebeat receiver")
		err := fb.beater.Run(fb.beat)
		if err != nil {
			fb.logger.Error("filebeat receiver run error", zap.Error(err))
		}
	}()
	return nil
}

func (fb *filebeatReceiver) Shutdown(ctx context.Context) error {
	fb.logger.Info("stopping filebeat receiver")
	fb.beater.Stop()
	fb.wg.Wait()
	return nil
}
