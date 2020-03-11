// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/boolexp"
)

// Program represents a program that must be started or must run.
type Program struct {
	Spec   Spec
	Config *transpiler.AST
}

// Cmd return the execution command to run.
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
func Programs(singleConfig *transpiler.AST) (map[string][]Program, error) {
	grouped, err := groupByOutputs(singleConfig)
	if err != nil {
		return nil, errors.New(err, errors.TypeConfig, "fail to extract program configuration")
	}

	groupedPrograms := make(map[string][]Program)
	for k, config := range grouped {
		programs, err := detectPrograms(config)
		if err != nil {
			return nil, errors.New(err, errors.TypeConfig, "fail to generate program configuration")
		}
		groupedPrograms[k] = programs
	}

	return groupedPrograms, nil
}

func detectPrograms(singleConfig *transpiler.AST) ([]Program, error) {
	programs := make([]Program, 0)
	for _, spec := range Supported {
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

func groupByOutputs(single *transpiler.AST) (map[string]*transpiler.AST, error) {
	const (
		outputsKey = "outputs"
		outputKey  = "output"
		streamsKey = "datasources"
		typeKey    = "type"
	)

	if _, found := transpiler.Select(single, outputsKey); !found {
		return nil, errors.New("invalid configuration missing outputs configuration")
	}

	// Normalize using an intermediate map.
	normMap, err := single.Map()
	if err != nil {
		return nil, errors.New(err, "could not read configuration")
	}

	// Recreates multiple configuration grouped by the name of the outputs.
	// Each configuration will be started into his own operator with the same name as the output.
	grouped := make(map[string]map[string]interface{})

	m, ok := normMap[outputsKey]
	if !ok {
		return nil, errors.New("fail to received a list of configured outputs")
	}

	out, ok := m.(map[string]interface{})
	if !ok {
		return nil, errors.New(fmt.Errorf(
			"invalid outputs configuration received, expecting a map not a %T",
			m,
		))
	}

	for k, v := range out {
		outputsOptions, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid type for output configuration block")
		}

		t, ok := outputsOptions[typeKey]
		if !ok {
			return nil, fmt.Errorf("missing output type named output %s", k)
		}

		n, ok := t.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type received %T and expecting a string", t)
		}

		delete(outputsOptions, typeKey)

		// Propagate global configuration to each individual configuration.
		clone := cloneMap(normMap)
		delete(clone, outputsKey)
		clone[outputKey] = map[string]interface{}{n: v}
		clone[streamsKey] = make([]map[string]interface{}, 0)

		grouped[k] = clone
	}

	s, ok := normMap[streamsKey]
	if !ok {
		s = make([]interface{}, 0)
	}

	list, ok := s.([]interface{})
	if !ok {
		return nil, errors.New("fail to receive a list of configured streams")
	}

	for _, item := range list {
		stream, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(
				"invalid type for stream expecting a map of options and received %T",
				item,
			)
		}
		targetName := findOutputName(stream)

		// Do we have configuration for that specific outputs if not we fail to load the configuration.
		config, ok := grouped[targetName]
		if !ok {
			return nil, fmt.Errorf("unknown configuration output with name %s", targetName)
		}

		streams := config[streamsKey].([]map[string]interface{})
		streams = append(streams, stream)

		config[streamsKey] = streams
		grouped[targetName] = config
	}

	transpiled := make(map[string]*transpiler.AST)

	for name, group := range grouped {
		if len(group[streamsKey].([]map[string]interface{})) == 0 {
			continue
		}

		ast, err := transpiler.NewAST(group)
		if err != nil {
			return nil, errors.New(err, "fail to generate configuration for output name %s", name)
		}

		transpiled[name] = ast
	}

	return transpiled, nil
}

func findOutputName(m map[string]interface{}) string {
	const (
		defaultOutputName = "default"
		useOutputKey      = "use_output"
	)

	output, ok := m[useOutputKey]
	if !ok {
		return defaultOutputName
	}

	return output.(string)
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		sV, ok := v.(map[string]interface{})
		if ok {
			newMap[k] = cloneMap(sV)
			continue
		}
		newMap[k] = v
	}

	return newMap
}
