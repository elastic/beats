// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/transpiler"
)

const (
	monitoringName            = "FLEET_MONITORING"
	programsKey               = "programs"
	monitoringKey             = "settings.monitoring"
	monitoringUseOutputKey    = "settings.monitoring.use_output"
	monitoringOutputFormatKey = "outputs.%s"
	outputKey                 = "output"

	enabledKey       = "settings.monitoring.enabled"
	outputsKey       = "outputs"
	elasticsearchKey = "elasticsearch"
	typeKey          = "type"
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
		// get monitoring output name to be used
		useOutputNode, found := transpiler.Lookup(rootAst, monitoringUseOutputKey)
		if !found {
			return programsToRun, nil
		}

		monitoringOutputNameKey, ok := useOutputNode.Value().(*transpiler.StrVal)
		if !ok {
			return programsToRun, nil
		}

		monitoringOutputName := monitoringOutputNameKey.String()

		ast := rootAst.Clone()
		if err := getMonitoringRule(monitoringOutputName).Apply(ast); err != nil {
			return programsToRun, err
		}

		config, err = ast.Map()
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

func getMonitoringRule(outputName string) *transpiler.RuleList {
	monitoringOutputSelector := fmt.Sprintf(monitoringOutputFormatKey, outputName)
	return transpiler.NewRuleList(
		transpiler.Copy(monitoringOutputSelector, outputKey),
		transpiler.Rename(fmt.Sprintf("%s.%s", outputsKey, outputName), elasticsearchKey),
		transpiler.Filter(monitoringKey, programsKey, outputKey),
	)
}
