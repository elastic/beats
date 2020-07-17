// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

const (
	monitoringName            = "FLEET_MONITORING"
	programsKey               = "programs"
	monitoringKey             = "agent.monitoring"
	monitoringUseOutputKey    = "agent.monitoring.use_output"
	monitoringOutputFormatKey = "outputs.%s"
	outputKey                 = "output"

	enabledKey        = "agent.monitoring.enabled"
	logsKey           = "agent.monitoring.logs"
	metricsKey        = "agent.monitoring.metrics"
	outputsKey        = "outputs"
	elasticsearchKey  = "elasticsearch"
	typeKey           = "type"
	defaultOutputName = "default"
)

func injectMonitoring(outputGroup string, rootAst *transpiler.AST, programsToRun []program.Program) ([]program.Program, error) {
	var err error
	monitoringProgram := program.Program{
		Spec: program.Spec{
			Name: monitoringName,
			Cmd:  monitoringName,
		},
	}

	config := make(map[string]interface{})
	// if monitoring is not specified use default one where everything is enabled
	if _, found := transpiler.Lookup(rootAst, monitoringKey); !found {
		monitoringNode := transpiler.NewDict([]transpiler.Node{
			transpiler.NewKey("enabled", transpiler.NewBoolVal(true)),
			transpiler.NewKey("logs", transpiler.NewBoolVal(true)),
			transpiler.NewKey("metrics", transpiler.NewBoolVal(true)),
			transpiler.NewKey("use_output", transpiler.NewStrVal("default")),
		})

		transpiler.Insert(rootAst, transpiler.NewKey("monitoring", monitoringNode), "settings")
	}

	// get monitoring output name to be used
	monitoringOutputName := defaultOutputName
	useOutputNode, found := transpiler.Lookup(rootAst, monitoringUseOutputKey)
	if found {
		monitoringOutputNameKey, ok := useOutputNode.Value().(*transpiler.StrVal)
		if !ok {
			return programsToRun, nil
		}

		monitoringOutputName = monitoringOutputNameKey.String()
	}

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
