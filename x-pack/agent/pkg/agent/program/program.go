// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"strings"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
	"github.com/elastic/beats/x-pack/agent/pkg/boolexp"
)

// Program represents a program that must be started or must run.
type Program struct {
	Spec   Spec
	Config *transpiler.AST
}

// Cmd return the exection command to run.
func (p *Program) Cmd() string {
	return p.Spec.Cmd
}

// Checksum return the checksum of the current instance of the program.
func (p *Program) Checksum() string {
	return p.Config.HashStr()
}

// Identifier returns the Program unique identifier.
func (p *Program) Identifier() string {
	return strings.ToLower(p.Spec.Name)
}

// Configuration return the program configuration in a map[string]iface format.
func (p *Program) Configuration() map[string]interface{} {
	m, err := p.Config.Map()
	if err != nil {
		// TODO, that should not panic, refactor to remove any panic.
		// Will refactor to never return an error at this stage.
		panic(err)
	}
	return m
}

// Programs take a Tree representation of the main configuration and apply all the different
// programs rules and generate individual configuration from the rules.
func Programs(singleConfig *transpiler.AST) ([]Program, error) {
	programs := make([]Program, 0)
	for _, spec := range Supported {
		// TODO: better error handling here.
		specificAST := singleConfig.Clone()
		err := spec.Rules.Apply(specificAST)
		if err != nil {
			return nil, err
		}

		if len(spec.When) == 0 {
			return nil, ErrMissingWhen
		}

		expression, err := boolexp.New(spec.When, methodsEnv(specificAST))
		if err != nil {
			return nil, err
		}

		ok, err := expression.Eval(&varStoreAST{ast: specificAST})
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		program := Program{
			Spec:   spec,
			Config: specificAST,
		}
		programs = append(programs, program)
	}
	return programs, nil
}

// KnownProgramNames returns a list of runnable programs by the agent.
func KnownProgramNames() []string {
	names := make([]string, len(Supported))
	for idx, program := range Supported {
		names[idx] = program.Name
	}
	return names
}
