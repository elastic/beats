// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	journalbeatCmd "github.com/elastic/beats/v7/journalbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	settings := journalbeatCmd.JournalbeatSettings()
	settings.ElasticLicensed = true
	RootCmd = journalbeatCmd.Initialize(settings)
}
