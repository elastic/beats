// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package abreceiver

import (
	"context"
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/component"
)

type auditbeatReceiver struct {
	xpInstance.BeatReceiver
}

func (ab *auditbeatReceiver) Start(ctx context.Context, host component.Host) error {
	ab.Logger.Info("starting auditbeat receiver")
	if err := ab.BeatReceiver.Start(host); err != nil {
		return fmt.Errorf("error starting auditbeat receiver: %w", err)
	}
	return nil
}

func (ab *auditbeatReceiver) Shutdown(ctx context.Context) error {
	ab.Logger.Info("stopping auditbeat receiver")
	if err := ab.BeatReceiver.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping auditbeat receiver: %w", err)
	}
	return nil
}
