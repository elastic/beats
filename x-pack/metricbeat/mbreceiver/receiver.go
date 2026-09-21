// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
)

type metricbeatReceiver struct {
	xpInstance.BeatReceiver
}

func (mb *metricbeatReceiver) Start(ctx context.Context, host component.Host) error {
	mb.Logger.Info("starting metricbeat receiver")
	if err := mb.BeatReceiver.Start(host); err != nil {
		return fmt.Errorf("error starting metricbeat receiver: %w", err)
	}

	return nil
}

func (mb *metricbeatReceiver) Shutdown(ctx context.Context) error {
	mb.Logger.Info("stopping metricbeat receiver")
	if err := mb.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	return nil
}
