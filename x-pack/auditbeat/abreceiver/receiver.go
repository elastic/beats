// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package abreceiver

import (
	"context"
	"fmt"
	"sync"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type auditbeatReceiver struct {
	xpInstance.BeatReceiver
	wg sync.WaitGroup
}

func (ab *auditbeatReceiver) Start(ctx context.Context, host component.Host) error {
	ab.wg.Add(1)
	go func() {
		defer ab.wg.Done()
		ab.Logger.Info("starting auditbeat receiver")
		if err := ab.BeatReceiver.Start(host); err != nil {
			ab.Logger.Error("error starting auditbeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (ab *auditbeatReceiver) Shutdown(ctx context.Context) error {
	ab.Logger.Info("stopping auditbeat receiver")
	if err := ab.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping auditbeat receiver: %w", err)
	}
	ab.wg.Wait()
	return nil
}
