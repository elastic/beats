// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"strings"

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

func emitter(log *logger.Logger, router programsDispatcher, modifiers *configModifiers, reloadables ...reloadable) emitterFunc {
	return func(c *config.Config) error {
		if err := InjectAgentConfig(c); err != nil {
			return err
		}

		log.Debug("Transforming configuration into a tree")
		m, err := c.ToMapStr()
		if err != nil {
			return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
		}

		ast, err := transpiler.NewAST(m)
		if err != nil {
			return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
		}

		for _, filter := range modifiers.Filters {
			if err := filter(log, ast); err != nil {
				return errors.New(err, "failed to filter configuration", errors.TypeConfig)
			}
		}

		log.Debugf("Supported programs: %s", strings.Join(program.KnownProgramNames(), ", "))
		log.Debug("Converting single configuration into specific programs configuration")

		programsToRun, err := program.Programs(ast)
		if err != nil {
			return err
		}

		for _, decorator := range modifiers.Decorators {
			for outputType, ptr := range programsToRun {
				programsToRun[outputType], err = decorator(outputType, ast, ptr)
				if err != nil {
					return err
				}
			}
		}

		for _, r := range reloadables {
			if err := r.Reload(c); err != nil {
				return err
			}
		}

		return router.Dispatch(ast.HashStr(), programsToRun)
	}
}

func readfiles(files []string, emitter emitterFunc) error {
	c, err := config.LoadFiles(files...)
	if err != nil {
		return errors.New(err, "could not load or merge configuration", errors.TypeConfig)
	}

	return emitter(c)
}
