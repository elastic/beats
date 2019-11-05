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

type decoratorFunc = func(*transpiler.AST, []program.Program) ([]program.Program, error)

func emitter(log *logger.Logger, router *router, decorators ...decoratorFunc) emitterFunc {
	return func(files []string) error {
		c, err := config.LoadFiles(files...)
		if err != nil {
			return errors.Wrap(err, "could not load or merge configuration")
		}

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
			programsToRun, err = decorator(ast, programsToRun)
			if err != nil {
				return err
			}
		}

		grouped := map[routingKey][]program.Program{
			defautlRK: programsToRun,
		}

		return router.Dispatch(ast.HashStr(), grouped)
	}
}
