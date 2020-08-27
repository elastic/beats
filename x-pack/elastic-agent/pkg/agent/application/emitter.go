// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

type decoratorFunc = func(string, *transpiler.AST, []program.Program) ([]program.Program, error)
type filterFunc = func(*logger.Logger, *transpiler.AST) error

type reloadable interface {
	Reload(cfg *config.Config) error
}

type configModifiers struct {
	Filters    []filterFunc
	Decorators []decoratorFunc
}

type programsDispatcher interface {
	Dispatch(id string, grpProg map[routingKey][]program.Program) error
}

type emitterController struct {
	logger      *logger.Logger
	controller  composable.Controller
	router      programsDispatcher
	modifiers   *configModifiers
	reloadables []reloadable

	// state
	lock   sync.RWMutex
	config *config.Config
	ast    *transpiler.AST
	vars   []composable.Vars
}

func (e *emitterController) Update(c *config.Config) error {
	if err := InjectAgentConfig(c); err != nil {
		return err
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
	for _, filter := range e.modifiers.Filters {
		if err := filter(e.logger, rawAst); err != nil {
			return errors.New(err, "failed to filter configuration", errors.TypeConfig)
		}
	}

	// sanitary check that nothing in the config is wrong when it comes to variable syntax
	ast := rawAst.Clone()
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		renderedInputs, err := renderInputs(inputs, []composable.Vars{
			{
				Mapping: map[string]interface{}{},
			},
		})
		if err != nil {
			return err
		}
		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			return err
		}
	}

	programsToRun, err := program.Programs(ast)
	if err != nil {
		return err
	}

	for _, decorator := range e.modifiers.Decorators {
		for outputType, ptr := range programsToRun {
			programsToRun[outputType], err = decorator(outputType, ast, ptr)
			if err != nil {
				return err
			}
		}
	}

	e.lock.Lock()
	e.config = c
	e.ast = rawAst
	e.lock.Unlock()

	return e.update()
}

func (e *emitterController) Set(vars []composable.Vars) {
	e.lock.Lock()
	ast := e.ast
	e.vars = vars
	e.lock.Unlock()

	if ast != nil {
		err := e.update()
		if err != nil {
			e.logger.Errorf("Failed to render new configuration with latest context from composable controller: %s", err)
		}
	}
}

func (e *emitterController) update() error {
	e.lock.RLock()
	cfg := e.config
	rawAst := e.ast
	varsArray := e.vars
	e.lock.RUnlock()

	ast := rawAst.Clone()
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		renderedInputs, err := renderInputs(inputs, varsArray)
		if err != nil {
			return err
		}
		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			return err
		}
	}

	e.logger.Debug("Converting single configuration into specific programs configuration")

	programsToRun, err := program.Programs(ast)
	if err != nil {
		return err
	}

	for _, decorator := range e.modifiers.Decorators {
		for outputType, ptr := range programsToRun {
			programsToRun[outputType], err = decorator(outputType, ast, ptr)
			if err != nil {
				return err
			}
		}
	}

	for _, r := range e.reloadables {
		if err := r.Reload(cfg); err != nil {
			return err
		}
	}

	return e.router.Dispatch(ast.HashStr(), programsToRun)
}

func emitter(ctx context.Context, log *logger.Logger, controller composable.Controller, router programsDispatcher, modifiers *configModifiers, reloadables ...reloadable) (emitterFunc, error) {
	log.Debugf("Supported programs: %s", strings.Join(program.KnownProgramNames(), ", "))

	ctrl := &emitterController{
		logger:      log,
		controller:  controller,
		router:      router,
		modifiers:   modifiers,
		reloadables: reloadables,
		vars: []composable.Vars{
			{
				Mapping: map[string]interface{}{},
			},
		},
	}
	err := controller.Run(ctx, func(vars []composable.Vars) {
		ctrl.Set(vars)
	})
	if err != nil {
		return nil, errors.New(err, "failed to start composable controller")
	}
	return func(c *config.Config) error {
		return ctrl.Update(c)
	}, nil
}

func readfiles(files []string, emitter emitterFunc) error {
	c, err := config.LoadFiles(files...)
	if err != nil {
		return errors.New(err, "could not load or merge configuration", errors.TypeConfig)
	}

	return emitter(c)
}

func renderInputs(inputs transpiler.Node, varsArray []composable.Vars) (transpiler.Node, error) {
	l, ok := inputs.Value().(*transpiler.List)
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}
	nodes := []transpiler.Node{}
	nodesMap := map[string]*transpiler.Dict{}
	for _, vars := range varsArray {
		for _, node := range l.Value().([]transpiler.Node) {
			dict, ok := node.Clone().(*transpiler.Dict)
			if !ok {
				continue
			}
			err := dict.Apply(vars)
			if err == composable.ErrNoMatch {
				// has a variable that didn't exist, so we ignore it
				continue
			}
			if err != nil {
				// another error that needs to be reported
				return nil, err
			}
			hash := string(dict.Hash())
			_, exists := nodesMap[hash]
			if !exists {
				nodesMap[hash] = dict
				nodes = append(nodes, dict)
			}
		}
	}
	return transpiler.NewList(nodes), nil
}
