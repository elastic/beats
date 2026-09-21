// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pbreceiver

import (
	"context"
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
)

type packetbeatReceiver struct {
	xpInstance.BeatReceiver
}

func (pb *packetbeatReceiver) Start(ctx context.Context, host component.Host) error {
	pb.Logger.Info("starting packetbeat receiver")
	if err := pb.BeatReceiver.Start(host); err != nil {
		return fmt.Errorf("error starting packetbeat receiver: %w", err)
	}
	return nil
}

func (pb *packetbeatReceiver) Shutdown(ctx context.Context) error {
	if err := pb.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping packetbeat receiver: %w", err)
	}
	return nil
}
