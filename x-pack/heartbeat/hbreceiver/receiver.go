// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
)

type heartbeatReceiver struct {
	xpInstance.BeatReceiver
	elasticsearchAuthRequester *esClient
}

func (hb *heartbeatReceiver) Start(_ context.Context, host component.Host) error {
	hb.Logger.Info("starting heartbeat receiver")

	if err := hb.BeatReceiver.Start(host); err != nil {
		return fmt.Errorf("starting heartbeat receiver: %w", err)
	}

	return nil
}

func (hb *heartbeatReceiver) Shutdown(ctx context.Context) error {
	hb.Logger.Info("stopping heartbeat receiver")
	defer func() {
		if hb.elasticsearchAuthRequester != nil {
			hb.elasticsearchAuthRequester.CloseIdleConnections()
		}
	}()
	if err := hb.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping heartbeat receiver: %w", err)
	}

	return nil
}
