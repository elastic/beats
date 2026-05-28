// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pbreceiver

import (
	"context"
	"fmt"
	"sync"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type packetbeatReceiver struct {
	xpInstance.BeatReceiver
	wg sync.WaitGroup
}

func (pb *packetbeatReceiver) Start(ctx context.Context, host component.Host) error {
	pb.wg.Add(1)
	go func() {
		defer pb.wg.Done()
		pb.Logger.Info("starting packetbeat receiver")
		if err := pb.BeatReceiver.Start(host); err != nil {
			pb.Logger.Error("error starting packetbeat receiver", zap.Error(err))
		}
	}()
	return nil
}

func (pb *packetbeatReceiver) Shutdown(ctx context.Context) error {
	pb.Logger.Info("stopping packetbeat receiver")
	if err := pb.BeatReceiver.Shutdown(); err != nil {
		return fmt.Errorf("error stopping packetbeat receiver: %w", err)
	}
	pb.wg.Wait()
	return nil
}
