// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatreceiver"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type metricbeatReceiver struct {
	beatreceiver.BeatReceiver
	wg sync.WaitGroup
}

func (mb *metricbeatReceiver) Start(ctx context.Context, host component.Host) error {
	mb.wg.Add(1)
	go func() {
		defer mb.wg.Done()
		mb.Logger.Info("starting metricbeat receiver")
		if err := mb.BeatReceiver.Start(); err != nil {
			mb.Logger.Error("error starting metricbeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (mb *metricbeatReceiver) Shutdown(ctx context.Context) error {
	mb.Logger.Info("stopping metricbeat receiver")
	if err := mb.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	mb.wg.Wait()
	return nil
}
