// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqreceiver

import (
	"context"
	"fmt"
	"sync"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type osquerybeatReceiver struct {
	xpInstance.BeatReceiver
	wg sync.WaitGroup
}

func (ob *osquerybeatReceiver) Start(ctx context.Context, host component.Host) error {
	ob.wg.Add(1)
	go func() {
		defer ob.wg.Done()
		ob.Logger.Info("starting osquerybeat receiver")
		if err := ob.BeatReceiver.Start(host); err != nil {
			ob.Logger.Error("error starting osquerybeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (ob *osquerybeatReceiver) Shutdown(ctx context.Context) error {
	ob.Logger.Info("stopping osquerybeat receiver")
	if err := ob.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping osquerybeat receiver: %w", err)
	}
	ob.wg.Wait()
	return nil
}
