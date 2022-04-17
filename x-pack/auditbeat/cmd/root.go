// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	auditbeatcmd "github.com/menderesk/beats/v7/auditbeat/cmd"
	"github.com/menderesk/beats/v7/libbeat/cmd"

	// Register Auditbeat x-pack modules.
	_ "github.com/menderesk/beats/v7/x-pack/auditbeat/include"
	_ "github.com/menderesk/beats/v7/x-pack/libbeat/include"
)

// Name of the beat
var Name = auditbeatcmd.Name

// RootCmd to handle beats CLI.
var RootCmd *cmd.BeatsRootCmd

func init() {
	settings := auditbeatcmd.AuditbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = auditbeatcmd.Initialize(settings)
}
