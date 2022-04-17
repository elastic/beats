// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	funcmd "github.com/menderesk/beats/v7/x-pack/functionbeat/function/cmd"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/manager/beater"
)

// Name of this beat
var Name = "functionbeat"

// RootCmd to handle functionbeat
var RootCmd *funcmd.FunctionCmd

func init() {
	RootCmd = funcmd.NewFunctionCmd(Name, beater.New)
}
