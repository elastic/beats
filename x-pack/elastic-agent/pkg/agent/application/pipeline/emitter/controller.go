// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package emitter

import (
	"context"
	"sync"

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

type reloadable interface {
	Reload(cfg *config.Config) error
}

// Controller is an emitter controller handling config updates.
type Controller struct {
	logger      *logger.Logger
	agentInfo   *info.AgentInfo
	controller  composable.Controller
	router      pipeline.Router
	modifiers   *pipeline.ConfigModifiers
	reloadables []reloadable
	caps        capabilities.Capability

	// state
	lock       sync.RWMutex
	updateLock sync.Mutex
	config     *config.Config
	ast        *transpiler.AST
	vars       []*transpiler.Vars
}

// NewController creates a new emitter controller.
func NewController(
	log *logger.Logger,
	agentInfo *info.AgentInfo,
	controller composable.Controller,
	router pipeline.Router,
	modifiers *pipeline.ConfigModifiers,
	caps capabilities.Capability,
	reloadables ...reloadable,
) *Controller {
	init, _ := transpiler.NewVars(map[string]interface{}{}, nil)

	return &Controller{
		logger:      log,
		agentInfo:   agentInfo,
		controller:  controller,
		router:      router,
		modifiers:   modifiers,
		reloadables: reloadables,
		vars:        []*transpiler.Vars{init},
		caps:        caps,
	}
}

// Update applies config change and performes all steps necessary to apply it.
func (e *Controller) Update(ctx context.Context, c *config.Config) (err error) {
	span, ctx := apm.StartSpan(ctx, "update", "app.internal")
	defer func() {
		apm.CaptureError(ctx, rErr).Send()
		span.End()
	}()

	if rErr = info.InjectAgentConfig(c); rErr != nil {
		return
	}

	// perform and verify ast translation
	m, err := c.ToMapStr()
	if err != nil {
		return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
	}

	rawAst, err := transpiler.NewAST(m)
	if err != nil {
		return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
	}

	if e.caps != nil {
		var ok bool
		updatedAst, err := e.caps.Apply(rawAst)
		if err != nil {
			rErr = errors.New(err, "failed to apply capabilities")
			return
		}

		rawAst, ok = updatedAst.(*transpiler.AST)
		if !ok {
			rErr = errors.New("failed to transform object returned from capabilities to AST", errors.TypeConfig)
			return
		}
	}

	for _, filter := range e.modifiers.Filters {
		if err := filter(e.logger, rawAst); err != nil {
			rErr = errors.New(err, "failed to filter configuration", errors.TypeConfig)
			return
		}
	}

	e.lock.Lock()
	e.config = c
	e.ast = rawAst
	e.lock.Unlock()

	return e.update(ctx)
}

// Set sets the transpiler vars for dynamic inputs resolution.
func (e *Controller) Set(ctx context.Context, vars []*transpiler.Vars) {
	var err error
	span, ctx := apm.StartSpan(ctx, "Set", "app.internal")
	defer func() {
		apm.CaptureError(ctx, err).Send()
		span.End()
	}()
	e.lock.Lock()
	ast := e.ast
	e.vars = vars
	e.lock.Unlock()

	if ast != nil {
		err = e.update(ctx)
		if err != nil {
			e.logger.Errorf("Failed to render configuration with latest context from composable controller: %s", err)
		}
	}
}

func (e *Controller) update(ctx context.Context) (err error) {
	span, ctx := apm.StartSpan(ctx, "update", "app.internal")
		apm.CaptureError(ctx, err).Send()
		span.End()
	}()
	// locking whole update because it can be called concurrently via Set and Update method
	e.updateLock.Lock()
	defer e.updateLock.Unlock()

	e.lock.RLock()
	cfg := e.config
	rawAst := e.ast
	varsArray := e.vars
	e.lock.RUnlock()

	ast := rawAst.Clone()
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		var renderedInputs transpiler.Node
		renderedInputs, err = transpiler.RenderInputs(inputs, varsArray)
		if err != nil {
			return err
		}
		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			return err
		}
	}

	e.logger.Debug("Converting single configuration into specific programs configuration")

	programsToRun := make(map[string][]program.Program)
	programsToRun, err = program.Programs(e.agentInfo, ast)
	if err != nil {
		return err
	}

	for _, decorator := range e.modifiers.Decorators {
		for outputType, ptr := range programsToRun {
			programsToRun[outputType], err = decorator(e.agentInfo, outputType, ast, ptr)
			if err != nil {
				return err
			}
		}
	}

	for _, r := range e.reloadables {
		if err = r.Reload(cfg); err != nil {
			return err
		}
	}

	return e.router.Route(ctx, ast.HashStr(), programsToRun)
}
