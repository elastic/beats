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
<<<<<<< HEAD
	ob.wg.Go(func() {
		ob.Logger.Info("starting osquerybeat receiver")
		if err := ob.BeatReceiver.Start(host); err != nil {
			ob.Logger.Error("error starting osquerybeat receiver", zap.Error(err))
		}
	})
=======
	ob.Logger.Info("starting osquerybeat receiver")
	if err := ob.BeatReceiver.Start(ctx, host); err != nil {
		return fmt.Errorf("error starting osquerybeat receiver: %w", err)
	}
>>>>>>> 36c8c4f (Correctly set useragent field in otel mode (#53654))
	return nil
}

func (ob *osquerybeatReceiver) Shutdown(ctx context.Context) error {
	ob.Logger.Info("stopping osquerybeat receiver")
	if err := ob.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping osquerybeat receiver: %w", err)
	}
	ob.wg.Wait()
	return nil
}
