// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

type decoratorFunc = func(string, *transpiler.AST, []program.Program) ([]program.Program, error)

func emitter(log *logger.Logger, router *router, decorators ...decoratorFunc) emitterFunc {
	return func(c *config.Config) error {
		log.Debug("Transforming configuration into a tree")
		m, err := c.ToMapStr()
		if err != nil {
			return errors.Wrap(err, "could not create the AST from the configuration")
		}

		ast, err := transpiler.NewAST(m)
		if err != nil {
			return errors.Wrap(err, "could not create the AST from the configuration")
		}

		log.Debugf("Supported programs: %s", strings.Join(program.KnownProgramNames(), ", "))
		log.Debug("Converting single configuration into specific programs configuration")

		programsToRun, err := program.Programs(ast)
		if err != nil {
			return err
		}

		for _, decorator := range decorators {
			for outputType, ptr := range programsToRun {
				programsToRun[outputType], err = decorator(outputType, ast, ptr)
				if err != nil {
					return err
				}
			}
		}

		return router.Dispatch(ast.HashStr(), programsToRun)
	}
}

func readfiles(files []string, emitter emitterFunc) error {
	c, err := config.LoadFiles(files...)
	if err != nil {
		return errors.Wrap(err, "could not load or merge configuration")
	}

	return emitter(c)
}
