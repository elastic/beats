// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/go-sysinfo/types"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func injectFleet(cfg *config.Config, hostInfo types.HostInfo, agentInfo *info.AgentInfo) func(*logger.Logger, *transpiler.AST) error {
	return func(logger *logger.Logger, rootAst *transpiler.AST) error {
		ecsMeta, err := agentInfo.ECSMetadata()
		if err != nil {
			return err
		}
		logLevel := ecsMeta.Elastic.Agent.LogLevel

		config, err := cfg.ToMapStr()
		if err != nil {
			return err
		}
		ast, err := transpiler.NewAST(config)
		if err != nil {
			return err
		}
		token, ok := transpiler.Lookup(ast, "fleet.access_api_key")
		if !ok {
			return fmt.Errorf("failed to get api key from fleet config")
		}

		kbn, ok := transpiler.Lookup(ast, "fleet.kibana")
		if !ok {
			return fmt.Errorf("failed to get kibana config key from fleet config")
		}

		agent, ok := transpiler.Lookup(ast, "agent")
		if !ok {
			return fmt.Errorf("failed to get agent key from fleet config")
		}

		if _, found := transpiler.Lookup(ast, "agent.logging.level"); !found {
			transpiler.Insert(ast, transpiler.NewKey("level", transpiler.NewStrVal(logLevel)), "agent.logging")
		}

		host := transpiler.NewKey("host", transpiler.NewDict([]transpiler.Node{
			transpiler.NewKey("id", transpiler.NewStrVal(hostInfo.UniqueID)),
		}))

		nodes := []transpiler.Node{agent, token, kbn, host}
		server, ok := transpiler.Lookup(ast, "fleet.server")
		if ok {
			nodes = append(nodes, server)
		}
		fleet := transpiler.NewDict(nodes)

		err = transpiler.Insert(rootAst, fleet, "fleet")
		if err != nil {
			return err
		}
		return nil
	}
}
