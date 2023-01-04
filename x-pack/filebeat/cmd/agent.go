// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func filebeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	procs := defaultProcessors()
	modules, err := management.CreateInputsFromStreams(rawIn, "logs", agentInfo, procs...)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}

	// Extract the module name from the stream-level type
	// these types are defined in the elastic-agent's specfiles
	for iter := range modules {
		if _, ok := modules[iter]["type"]; !ok {
			modules[iter]["type"] = rawIn.Type
		}
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config: %w", err)
	}

	return configList, nil
}

func defaultProcessors() []mapstr.M {
	// processors:
	// - add_host_metadata:
	// 	when.not.contains.tags: forwarded
	// - add_cloud_metadata: ~
	// - add_docker_metadata: ~
	// - add_kubernetes_metadata: ~
	return []mapstr.M{
		{
			"add_host_metadata": mapstr.M{
				"when.not.contains.tags": "forwarded",
			},
		},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{"add_kubernetes_metadata": nil},
	}
}
