// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	packetbeatCmd "github.com/elastic/beats/v7/packetbeat/cmd"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"

	// This registers the Npcap installer on Windows.
	_ "github.com/elastic/beats/v7/x-pack/packetbeat/npcap"
)

// Name of this beat.
var Name = packetbeatCmd.Name

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// packetbeatCfg is a callback registered via SetTransform that returns a packetbeat Elastic Agent client.Unit
// configuration generated from a raw Elastic Agent config
func packetbeatCfg(rawIn *proto.UnitExpectedConfig, _ *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	uconfig, err := conf.NewConfigFrom(rawIn.Source.AsMap())
	if err != nil {
		return nil, fmt.Errorf("error in conversion to conf.C: %w", err)
	}
	return []*reload.ConfigWithMeta{{Config: uconfig}}, nil
}

func init() {
	// Register packetbeat with central management to perform any needed config
	// transformations before agent configs are sent to the beat during reload.
	management.ConfigTransform.SetTransform(packetbeatCfg)
	settings := packetbeatCmd.PacketbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = packetbeatCmd.Initialize(settings)
}
