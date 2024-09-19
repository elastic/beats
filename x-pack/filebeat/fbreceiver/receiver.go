// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"

	"go.opentelemetry.io/collector/component"
)

type filebeatReceiver struct {
	beat   *beat.Beat
	beater beat.Beater
}

func (fb *filebeatReceiver) Start(ctx context.Context, host component.Host) error {
	go func() {
		fb.beater.Run(fb.beat)
	}()
	return nil
}

func (fb *filebeatReceiver) Shutdown(ctx context.Context) error {
	fb.beater.Stop()
	return nil
}
