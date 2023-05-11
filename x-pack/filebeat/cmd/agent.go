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
)

func filebeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	var modules []map[string]interface{}
	var err error
	if rawIn.Type == "shipper" { // place filebeat in "shipper mode", with one filebeat input per agent config input
		modules, err = management.CreateShipperInput(rawIn, "logs", agentInfo)
		if err != nil {
			return nil, fmt.Errorf("error creating shipper config from raw expected config: %w", err)
		}
	} else {
		modules, err = management.CreateInputsFromStreams(rawIn, "logs", agentInfo)
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
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config: %w", err)
	}

	return configList, nil
}
