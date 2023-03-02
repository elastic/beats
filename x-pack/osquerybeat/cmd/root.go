// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/spf13/cobra"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/beater"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
)

// Name of this beat
const (
	Name = "osquerybeat"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

var RootCmd = Osquerybeat()

func Osquerybeat() *cmd.BeatsRootCmd {
	management.ConfigTransform.SetTransform(osquerybeatCfg)
	settings := instance.Settings{
		Name:            Name,
		Processing:      processing.MakeDefaultSupport(true, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
		ElasticLicensed: true,
	}
	command := cmd.GenRootCmdWithSettings(beater.New, settings)

	// Add verify command
	command.AddCommand(genVerifyCmd(settings))

	return command
}

func genVerifyCmd(_ instance.Settings) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify installation",
		Run: cli.RunWith(
			func(_ *cobra.Command, args []string) error {
				log := logp.NewLogger("osquerybeat")
				err := install.VerifyWithExecutableDirectory(log)
				if err != nil {
					return err
				}
				return nil
			}),
	}
}

func osquerybeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	// Convert to streams, osquerybeat doesn't use streams
	streams := make([]*proto.Stream, 1)

	// Enforce the datastream dataset and type because the libbeat call to CreateInputsFromStreams
	// provides it's own defaults that are breaking the osquery with logstash
	// The target datastream for the publisher is expected to be logs-osquery_manager.result-<namespace>
	// while the libebeat management.CreateInputsFromStreams defaults to osquery-generic-default
	var datastream *proto.DataStream
	if rawIn.GetDataStream() != nil {
		// Copy by value and modify dataset and type
		ds := *rawIn.GetDataStream()
		ds.Dataset = config.DefaultDataset
		ds.Type = config.DefaultType
		datastream = &ds
	}

	streams[0] = &proto.Stream{
		Source:     rawIn.GetSource(),
		Id:         rawIn.GetId(),
		DataStream: datastream,
	}

	rawIn.Streams = streams

	procs := defaultProcessors()

	modules, err := management.CreateInputsFromStreams(rawIn, "osquery", agentInfo, procs...)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}
	for iter := range modules {
		modules[iter]["type"] = "log"
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}
	return configList, nil
}

func defaultProcessors() []mapstr.M {
	// 	processors:
	//   - add_host_metadata: ~
	//   - add_cloud_metadata: ~
	return []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
	}
}
