// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/logp"

	// register central management
	"github.com/elastic/beats/x-pack/libbeat/licenser"
	_ "github.com/elastic/beats/x-pack/libbeat/management"
)

const licenseDebugK = "license"

// AddXPack extends the given root folder with XPack features
func AddXPack(root *cmd.BeatsRootCmd, name string) {
	licenser.Enforce(logp.NewLogger(licenseDebugK), licenser.BasicAndAboveOrTrial)
	root.AddCommand(genEnrollCmd(name, ""))
}
