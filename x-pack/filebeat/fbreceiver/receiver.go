// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatreceiver"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type filebeatReceiver struct {
	beatreceiver.BeatReceiver
	wg sync.WaitGroup
}

func (fb *filebeatReceiver) Start(ctx context.Context, host component.Host) error {
	fb.wg.Add(1)
	go func() {
		defer fb.wg.Done()
		fb.Logger.Info("starting filebeat receiver")
		if err := fb.BeatReceiver.Start(); err != nil {
			fb.Logger.Error("error starting filebeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (fb *filebeatReceiver) Shutdown(ctx context.Context) error {
	fb.Logger.Info("stopping filebeat receiver")
	if err := fb.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping filebeat receiver: %w", err)
	}
	fb.wg.Wait()
	return nil
}
