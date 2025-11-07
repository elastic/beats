// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	heartbeatCmd "github.com/elastic/beats/v7/heartbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	_ "github.com/elastic/beats/v7/heartbeat/include"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// heartbeatCfg is a callback registered via SetTransform that returns a Elastic Agent client.Unit
// configuration generated from a raw Elastic Agent config
func heartbeatCfg(rawIn *proto.UnitExpectedConfig, _ *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	configList, err := management.CreateReloadConfigFromInputs(TransformRawIn(rawIn))
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config: %w", err)
	}
	return configList, nil
}

// TransformRawIn removes unwanted fields to keep consistent hashing on reload()
func TransformRawIn(rawIn *proto.UnitExpectedConfig) []map[string]interface{} {
	rawInput := []map[string]interface{}{rawIn.GetSource().AsMap()}

	for _, p := range rawInput {
		delete(p, "policy")
	}

	return rawInput
}

func init() {
	settings := heartbeatCmd.HeartbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = heartbeatCmd.Initialize(settings)
	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		management.ConfigTransform.SetTransform(heartbeatCfg)
	}
}
