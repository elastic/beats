// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/elastic/beats/libbeat/cmd"

	// register central management
	"github.com/elastic/beats/x-pack/libbeat/licenser"
	_ "github.com/elastic/beats/x-pack/libbeat/management"

	// register autodiscover providers
	_ "github.com/elastic/beats/x-pack/libbeat/autodiscover/providers/aws/ec2"
	_ "github.com/elastic/beats/x-pack/libbeat/autodiscover/providers/aws/elb"
)

// AddXPack extends the given root folder with XPack features
func AddXPack(root *cmd.BeatsRootCmd, name string) {
	licenser.Enforce(name, licenser.BasicAndAboveOrTrial)
	root.AddCommand(genEnrollCmd(name, ""))
}
