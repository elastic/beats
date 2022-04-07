// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	heartbeatCmd "github.com/elastic/beats/v8/heartbeat/cmd"
	"github.com/elastic/beats/v8/libbeat/cmd"

	_ "github.com/elastic/beats/v8/x-pack/libbeat/include"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	settings := heartbeatCmd.HeartbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = heartbeatCmd.Initialize(settings)
}
