// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"
	"sync"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type heartbeatReceiver struct {
	xpInstance.BeatReceiver
	wg sync.WaitGroup
}

func (hb *heartbeatReceiver) Start(ctx context.Context, host component.Host) error {
	hb.wg.Add(1)
	go func() {
		defer hb.wg.Done()
		hb.Logger.Info("starting heartbeat receiver")
		if err := hb.BeatReceiver.Start(host); err != nil {
			hb.Logger.Error("error starting heartbeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (hb *heartbeatReceiver) Shutdown(ctx context.Context) error {
	hb.Logger.Info("stopping heartbeat receiver")
	if err := hb.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping heartbeat receiver: %w", err)
	}
	hb.wg.Wait()
	return nil
}
