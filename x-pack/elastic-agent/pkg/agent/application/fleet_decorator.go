// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func injectFleet(cfg *config.Config) func(*logger.Logger, *transpiler.AST) error {
	return func(logger *logger.Logger, rootAst *transpiler.AST) error {
		config, err := cfg.ToMapStr()
		if err != nil {
			return err
		}
		ast, err := transpiler.NewAST(config)
		if err != nil {
			return err
		}
		api, ok := transpiler.Lookup(ast, "api")
		if !ok {
			return fmt.Errorf("failed to get api from fleet config")
		}
		agentInfo, ok := transpiler.Lookup(ast, "agent_info")
		if !ok {
			return fmt.Errorf("failed to get agent_info from fleet config")
		}
		fleet := transpiler.NewDict([]transpiler.Node{agentInfo, api})
		err = transpiler.Insert(rootAst, fleet, "fleet")
		if err != nil {
			return err
		}
		return nil
	}
}
