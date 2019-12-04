// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

const (
	monitoringName = "FLEET_MONITORING"
	programsKey    = "programs"
	monitoringKey  = "monitoring"
	enabledKey     = "monitoring.enabled"
	outputKey      = "output"
	outputsKey     = "outputs"
	typeKey        = "type"
)

func injectMonitoring(outputGroup string, rootAst *transpiler.AST, programsToRun []program.Program) ([]program.Program, error) {
	var err error
	monitoringProgram := program.Program{
		Spec: program.Spec{
			Name: monitoringName,
			Cmd:  monitoringName,
		},
	}

	var config map[string]interface{}

	if _, found := transpiler.Lookup(rootAst, monitoringKey); !found {
		config = make(map[string]interface{})
		config[enabledKey] = false
	} else {
		ast := rootAst.Clone()
		if err := getMonitoringRule(outputGroup).Apply(ast); err != nil {
			return programsToRun, err
		}

		config, err = ast.Map()
		if err != nil {
			return programsToRun, err
		}

		config, err = renameConfigOutput(config, outputGroup)
		if err != nil {
			return programsToRun, err
		}

		programList := make([]string, 0, len(programsToRun))
		for _, p := range programsToRun {
			programList = append(programList, p.Spec.Cmd)
		}
		// making program list part of the config
		// so it will get regenerated with every change
		config[programsKey] = programList
	}

	monitoringProgram.Config, err = transpiler.NewAST(config)
	if err != nil {
		return programsToRun, err
	}

	return append(programsToRun, monitoringProgram), nil
}

func renameConfigOutput(cfg map[string]interface{}, outputGroup string) (map[string]interface{}, error) {
	outputType, err := getTypeName(cfg, outputGroup)
	if err != nil {
		return nil, err
	}

	output, found := cfg[outputsKey]
	if !found {
		return nil, fmt.Errorf("output configuration not found for monitoring output group '%s'", outputType)
	}

	outputsMap, ok := output.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("output configuration is not a map for monitoring output group '%s'", outputType)
	}

	specificOutputCfg, found := outputsMap[outputGroup]
	if !found {
		return nil, fmt.Errorf("output type '%s' is not found when compiling monitoring configuration for monitoring output group '%s'", outputGroup, outputGroup)
	}

	delete(cfg, outputsKey)
	cfg[outputKey] = map[string]interface{}{
		outputType: specificOutputCfg,
	}

	return cfg, nil
}

func getTypeName(cfg map[string]interface{}, outputGroup string) (string, error) {
	output, found := cfg[outputsKey]
	if !found {
		return "", fmt.Errorf("output configuration not found for monitoring output group '%s'", outputGroup)
	}

	outputMap, ok := output.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("output configuration is not a map for monitoring output group '%s'", outputGroup)
	}

	specificOutput, found := outputMap[outputGroup]
	if !found {
		return "", fmt.Errorf("specific output configuration not found for monitoring output group '%s'", outputGroup)
	}

	specificOutputMap, ok := specificOutput.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("specific output configuration is not a map for monitoring output group '%s'", outputGroup)
	}

	typeName, found := specificOutputMap[typeKey]
	if !found {
		return "", fmt.Errorf("output type config option not found for monitoring output group '%s'", outputGroup)
	}

	typeStr, ok := typeName.(string)
	if !ok {
		return "", fmt.Errorf("output type configuration option is not a string for monitoring output group '%s'", outputGroup)
	}

	return typeStr, nil
}

func getMonitoringRule(outputName string) *transpiler.RuleList {
	return transpiler.NewRuleList(
		transpiler.Filter(monitoringKey, programsKey, outputsKey),
	)
}
