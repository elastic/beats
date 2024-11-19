// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"

	"go.opentelemetry.io/collector/component"
)

type metricbeatReceiver struct {
	beat   *beat.Beat
	beater beat.Beater
}

func (mb *metricbeatReceiver) Start(ctx context.Context, host component.Host) error {
	go func() {
		_ = mb.beater.Run(mb.beat)
	}()
	return nil
}

func (mb *metricbeatReceiver) Shutdown(ctx context.Context) error {
	mb.beater.Stop()
	return nil
}
