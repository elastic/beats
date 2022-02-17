// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package emitter

import (
	"context"
	"strings"

	"go.elastic.co/apm"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/capabilities"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// New creates a new emitter function.
func New(ctx context.Context, log *logger.Logger, agentInfo *info.AgentInfo, controller composable.Controller, router pipeline.Router, modifiers *pipeline.ConfigModifiers, caps capabilities.Capability, reloadables ...reloadable) (pipeline.EmitterFunc, error) {
	log.Debugf("Supported programs: %s", strings.Join(program.KnownProgramNames(), ", "))

	ctrl := NewController(log, agentInfo, controller, router, modifiers, caps, reloadables...)
	err := controller.Run(ctx, func(vars []*transpiler.Vars) {
		ctrl.Set(ctx, vars)
	})
	if err != nil {
		return nil, errors.New(err, "failed to start composable controller")
	}
	return func(ctx context.Context, c *config.Config) (err error) {
		span, ctx := apm.StartSpan(ctx, "update", "app.internal")
		defer func() {
			apm.CaptureError(ctx, err).Send()
			span.End()
		}()
		return ctrl.Update(ctx, c)
	}, nil
}
