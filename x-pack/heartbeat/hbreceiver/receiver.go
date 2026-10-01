// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
)

type heartbeatReceiver struct {
	xpInstance.BeatReceiver
	ctx                        context.Context
	cancel                     context.CancelFunc
	elasticsearchAuthRequester *eslegclient.Connection
}

func newHeartbeatReceiver(br xpInstance.BeatReceiver) *heartbeatReceiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &heartbeatReceiver{
		BeatReceiver: br,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (hb *heartbeatReceiver) Start(_ context.Context, host component.Host) error {
	hb.Logger.Info("starting heartbeat receiver")

	if err := hb.BeatReceiver.Start(host); err != nil {
		return fmt.Errorf("starting heartbeat receiver: %w", err)
	}

	return nil
}

func (hb *heartbeatReceiver) Shutdown(ctx context.Context) (err error) {
	hb.Logger.Info("stopping heartbeat receiver")
	if hb.cancel != nil {
		hb.cancel()
	}

	defer func() {
		if hb.elasticsearchAuthRequester != nil {
			if closeErr := hb.elasticsearchAuthRequester.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("closing Elasticsearch client: %w", closeErr))
			}
		}
	}()

	if err := hb.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping heartbeat receiver: %w", err)
	}

	return nil
}
