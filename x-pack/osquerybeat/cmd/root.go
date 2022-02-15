// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/beater"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
<<<<<<< HEAD
=======
	"github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/logp"
>>>>>>> 1c68693c14 (Osquerybeat: Add install verification for osquerybeat (#30388))
	"github.com/elastic/beats/v7/libbeat/publisher/processing"

	"github.com/spf13/cobra"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
<<<<<<< HEAD
=======
	"github.com/elastic/beats/v7/x-pack/osquerybeat/beater"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
>>>>>>> 1c68693c14 (Osquerybeat: Add install verification for osquerybeat (#30388))
)

// Name of this beat
const (
	Name = "osquerybeat"

	// ecsVersion specifies the version of ECS that this beat is implementing.
	ecsVersion = "1.12.0"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

var RootCmd = Osquerybeat()

func Osquerybeat() *cmd.BeatsRootCmd {
	settings := instance.Settings{
		Name:            Name,
		Processing:      processing.MakeDefaultSupport(true, withECSVersion, processing.WithAgentMeta()),
		ElasticLicensed: true,
	}
	command := cmd.GenRootCmdWithSettings(beater.New, settings)

	// Add verify command
	command.AddCommand(genVerifyCmd(settings))

	return command
}

func genVerifyCmd(settings instance.Settings) *cobra.Command {
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
