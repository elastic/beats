// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
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
	vars   []*transpiler.Vars
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

	e.lock.Lock()
	e.config = c
	e.ast = rawAst
	e.lock.Unlock()

	return e.update()
}

func (e *emitterController) Set(vars []*transpiler.Vars) {
	e.lock.Lock()
	ast := e.ast
	e.vars = vars
	e.lock.Unlock()

	if ast != nil {
		err := e.update()
		if err != nil {
			e.logger.Errorf("Failed to render configuration with latest context from composable controller: %s", err)
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

	init, _ := transpiler.NewVars(map[string]interface{}{})
	ctrl := &emitterController{
		logger:      log,
		controller:  controller,
		router:      router,
		modifiers:   modifiers,
		reloadables: reloadables,
		vars:        []*transpiler.Vars{init},
	}
	err := controller.Run(ctx, func(vars []*transpiler.Vars) {
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

func renderInputs(inputs transpiler.Node, varsArray []*transpiler.Vars) (transpiler.Node, error) {
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
			n, err := dict.Apply(vars)
			if err == transpiler.ErrNoMatch {
				// has a variable that didn't exist, so we ignore it
				continue
			}
			if err != nil {
				// another error that needs to be reported
				return nil, err
			}
			if n == nil {
				// condition removed it
				continue
			}
			dict = n.(*transpiler.Dict)
			dict = promoteProcessors(dict)
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

func promoteProcessors(dict *transpiler.Dict) *transpiler.Dict {
	p := dict.Processors()
	if p == nil {
		return dict
	}
	current, ok := dict.Find("processors")
	currentList, isList := current.Value().(*transpiler.List)
	if !isList {
		return dict
	}
	ast, _ := transpiler.NewAST(map[string]interface{}{
		"processors": p,
	})
	procs, _ := transpiler.Lookup(ast, "processors")
	nodes := nodesFromList(procs.Value().(*transpiler.List))
	if ok {
		nodes = append(nodes, nodesFromList(currentList)...)
	}
	dictNodes := dict.Value().([]transpiler.Node)
	set := false
	for i, node := range dictNodes {
		switch n := node.(type) {
		case *transpiler.Key:
			if n.Name() == "processors" {
				dictNodes[i] = transpiler.NewKey("processors", transpiler.NewList(nodes))
				set = true
			}
		}
		if set {
			break
		}
	}
	if !set {
		dictNodes = append(dictNodes, transpiler.NewKey("processors", transpiler.NewList(nodes)))
	}
	return transpiler.NewDict(dictNodes)
}

func nodesFromList(list *transpiler.List) []transpiler.Node {
	return list.Value().([]transpiler.Node)
}
