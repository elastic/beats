// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modifiers

import (
	"github.com/elastic/go-sysinfo/types"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// InjectFleet injects fleet metadata into a configuration.
func InjectFleet(cfg *config.Config, hostInfo types.HostInfo, agentInfo *info.AgentInfo) func(*logger.Logger, *transpiler.AST) error {
	return func(logger *logger.Logger, rootAst *transpiler.AST) error {
		config, err := cfg.ToMapStr()
		if err != nil {
			return err
		}
		ast, err := transpiler.NewAST(config)
		if err != nil {
			return err
		}
		fleet, ok := transpiler.Lookup(ast, "fleet")
		if !ok {
			// no fleet from configuration; skip
			return nil
		}

		// copy top-level agent.* into fleet.agent.* (this gets sent to Applications in this structure)
		if agent, ok := transpiler.Lookup(ast, "agent"); ok {
			if err := transpiler.Insert(ast, agent, "fleet"); err != nil {
				return errors.New(err, "inserting agent info failed")
			}
		}

		// ensure that the agent.logging.level is present
		if _, found := transpiler.Lookup(ast, "agent.logging.level"); !found {
			transpiler.Insert(ast, transpiler.NewKey("level", transpiler.NewStrVal(agentInfo.LogLevel())), "agent.logging")
		}

		// fleet.host to Agent can be the host to connect to Fleet Server, but to Applications it should
		// be the fleet.host.id. move fleet.host to fleet.hosts if fleet.hosts doesn't exist
		if _, ok := transpiler.Lookup(ast, "fleet.hosts"); !ok {
			if host, ok := transpiler.Lookup(ast, "fleet.host"); ok {
				if key, ok := host.(*transpiler.Key); ok {
					if value, ok := key.Value().(*transpiler.StrVal); ok {
						hosts := transpiler.NewList([]transpiler.Node{transpiler.NewStrVal(value.String())})
						if err := transpiler.Insert(ast, hosts, "fleet.hosts"); err != nil {
							return errors.New(err, "inserting fleet hosts failed")
						}
					}
				}
			}
		}

		// inject host.* into fleet.host.* (this gets sent to Applications in this structure)
		host := transpiler.NewKey("host", transpiler.NewDict([]transpiler.Node{
			transpiler.NewKey("id", transpiler.NewStrVal(hostInfo.UniqueID)),
		}))
		if err := transpiler.Insert(ast, host, "fleet"); err != nil {
			return errors.New(err, "inserting list of hosts failed")
		}

		// inject fleet.* from local AST to the rootAST so its present when sending to Applications.
		err = transpiler.Insert(rootAst, fleet.Value().(transpiler.Node), "fleet")
		if err != nil {
			return errors.New(err, "inserting fleet info failed")
		}
		return nil
	}
}
