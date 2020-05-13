// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func injectPreferV2Template(
	_ string,
	_ *transpiler.AST,
	programsToRun []program.Program,
) ([]program.Program, error) {

	const (
		outputKey              = "output"
		elasticsearchKey       = "elasticsearch"
		paramsKey              = "parameters"
		elasticsearchOutputKey = outputKey + "." + elasticsearchKey
	)

	params := common.MapStr{
		"output": common.MapStr{
			"elasticsearch": common.MapStr{
				"parameters": common.MapStr{
					"prefer_v2_templates": true,
				},
			},
		},
	}

	programList := make([]program.Program, 0, len(programsToRun))

	for _, program := range programsToRun {
		if _, found := transpiler.Lookup(program.Config, elasticsearchOutputKey); !found {
			programList = append(programList, program)
			continue
		}

		m, err := program.Config.Map()
		if err != nil {
			return programsToRun, err
		}

		// Add prefer_v2_templates to every bulk request on the elasticsearch output.
		// without this Elasticsearch will fallback to v1 templates even if the index matches existing v2.
		mStr := common.MapStr(m)
		mStr.DeepUpdate(params)

		a, err := transpiler.NewAST(mStr)
		if err != nil {
			return programsToRun, err
		}

		program.Config = a

		programList = append(programList, program)
	}

	return programList, nil
}
