// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// InjectAgentConfig injects config to a provided configuration.
func InjectAgentConfig(c *config.Config) error {
	globalConfig := agentGlobalConfig()
	if err := c.Merge(globalConfig); err != nil {
		return errors.New("failed to inject agent global config", err, errors.TypeConfig)
	}

	return nil
}

// agentGlobalConfig gets global config used for resolution of variables inside configuration
// such as ${path.data}.
func agentGlobalConfig() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"data":   paths.Data(),
			"config": paths.Config(),
			"home":   paths.Home(),
			"logs":   paths.Logs(),
		},
	}
}
