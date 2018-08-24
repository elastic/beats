// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import "github.com/elastic/beats/libbeat/cmd"

// AddXPack extends the given root folder with XPack features
func AddXPack(root *cmd.BeatsRootCmd, name string) {
	root.AddCommand(genEnrollCmd(name, ""))
}
