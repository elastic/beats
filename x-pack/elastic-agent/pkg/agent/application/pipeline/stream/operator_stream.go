// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stream

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

type operatorStream struct {
	configHandler pipeline.ConfigHandler
	log           *logger.Logger
}

type stater interface {
	State() map[string]state.State
}

type specer interface {
	Specs() map[string]program.Spec
}

func (b *operatorStream) Close() error {
	return b.configHandler.Close()
}

func (b *operatorStream) State() map[string]state.State {
	if s, ok := b.configHandler.(stater); ok {
		return s.State()
	}

	return nil
}

func (b *operatorStream) Specs() map[string]program.Spec {
	if s, ok := b.configHandler.(specer); ok {
		return s.Specs()
	}
	return nil
}

func (b *operatorStream) Execute(ctx context.Context, cfg configrequest.Request) (err error) {
	// span, ctx := apm.StartSpan(ctx, "route", "app.internal")
	// defer func() {
	// 	if err != nil {
	// 		apm.CaptureError(ctx, err).Send()
	// 	}
	// 	span.End()
	// }()
	err = b.configHandler.HandleConfig(ctx, cfg)
	return
}

func (b *operatorStream) Shutdown() {
	b.configHandler.Shutdown()
}
