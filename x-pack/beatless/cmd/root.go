// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/x-pack/beatless/beater"
)

// Name of this beat
var Name = "beatless"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", beater.New)
