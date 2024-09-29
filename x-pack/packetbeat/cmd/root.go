// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/processors"
	packetbeatCmd "github.com/elastic/beats/v7/packetbeat/cmd"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"

	// This registers the Npcap installer on Windows.
	_ "github.com/elastic/beats/v7/x-pack/packetbeat/npcap"

	// Enable pipelines.
	_ "github.com/elastic/beats/v7/x-pack/packetbeat/module"
)

// Name of this beat.
var Name = packetbeatCmd.Name

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// packetbeatCfg is a callback registered via SetTransform that returns a packetbeat Elastic Agent client.Unit
// configuration generated from a raw Elastic Agent config
func packetbeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	//grab and properly format the input streams
	inputStreams, err := management.CreateInputsFromStreams(rawIn, "logs", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error generating new stream config: %w", err)
	}

	// Packetbeat does its own transformations,
	// so update the existing config with our new transformations,
	// then send to packetbeat
	souceMap := rawIn.Source.AsMap()
	souceMap["streams"] = inputStreams

	uconfig, err := conf.NewConfigFrom(souceMap)
	if err != nil {
		return nil, fmt.Errorf("error in conversion to conf.C: %w", err)
	}
	return []*reload.ConfigWithMeta{{Config: uconfig}}, nil
}

func init() {
	globalProcs, err := processors.NewPluginConfigFromList(defaultProcessors())
	if err != nil { // these are hard-coded, shouldn't fail
		panic(fmt.Errorf("error creating global processors: %w", err))
	}
	settings := packetbeatCmd.PacketbeatSettings(globalProcs)
	settings.ElasticLicensed = true
	RootCmd = packetbeatCmd.Initialize(settings)
	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Register packetbeat with central management to perform any needed config
		// transformations before agent configs are sent to the beat during reload.
		management.ConfigTransform.SetTransform(packetbeatCfg)
	}
}

func defaultProcessors() []mapstr.M {
	// 	processors:
	//   - # Add forwarded to tags when processing data from a network tap or mirror.
	//     if.contains.tags: forwarded
	//     then:
	//       - drop_fields:
	//           fields: [host]
	//     else:
	//       - add_host_metadata: ~
	//   - add_cloud_metadata: ~
	//   - add_docker_metadata: ~
	//   - detect_mime_type:
	//       field: http.request.body.content
	//       target: http.request.mime_type
	//   - detect_mime_type:
	//       field: http.response.body.content
	//       target: http.response.mime_type
	return []mapstr.M{
		{
			"if.contains.tags": "forwarded",
			"then": []interface{}{
				mapstr.M{
					"drop_fields": mapstr.M{
						"fields": []interface{}{"host"},
					},
				},
			},
			"else": []interface{}{
				mapstr.M{
					"add_host_metadata": nil,
				},
			},
		},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{
			"detect_mime_type": mapstr.M{
				"field":  "http.request.body.content",
				"target": "http.request.mime_type",
			},
		},
		{
			"detect_mime_type": mapstr.M{
				"field":  "http.response.body.content",
				"target": "http.response.mime_type",
			},
		},
	}
}
