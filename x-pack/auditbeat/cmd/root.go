// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"strings"

	auditbeatcmd "github.com/elastic/beats/v7/auditbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"

	// Register Auditbeat x-pack modules.
	_ "github.com/elastic/beats/v7/x-pack/auditbeat/include"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
)

// Name of the beat
var Name = auditbeatcmd.Name

// RootCmd to handle beats CLI.
var RootCmd *cmd.BeatsRootCmd

// auditbeatCfg is a callback registered with central management to perform any needed config transformations
// before agent configs are sent to a beat
func auditbeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	procs := defaultProcessors()
	modules, err := management.CreateInputsFromStreams(rawIn, "logs", agentInfo, procs...)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}

	// Extract the type field that has "audit/auditd", treat this
	// as the module config key
	module := strings.Split(rawIn.Type, "/")[1]

	for iter := range modules {
		modules[iter]["module"] = module
	}

	// Format for the reloadable list needed bythe cm.Reload() method.
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config: %w", err)
	}

	return configList, nil
}

func init() {
	management.ConfigTransform.SetTransform(auditbeatCfg)
	settings := auditbeatcmd.AuditbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = auditbeatcmd.Initialize(settings)
}

func defaultProcessors() []mapstr.M {
	// 	processors:
	//   - add_host_metadata: ~
	//   - add_cloud_metadata: ~
	//   - add_docker_metadata: ~
	return []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
	}
}
