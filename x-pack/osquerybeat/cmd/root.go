// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/beater"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
)

// Name of this beat
var Name = "osquerybeat"

var RootCmd = Osquerybeat()

func Osquerybeat() *cmd.BeatsRootCmd {
	settings := instance.Settings{
		Name:            Name,
		ElasticLicensed: true,
	}
	command := cmd.GenRootCmdWithSettings(beater.New, settings)

	return command
}
