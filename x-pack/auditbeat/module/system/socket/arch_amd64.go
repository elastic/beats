// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package socket

import "github.com/elastic/beats/v7/libbeat/common"

var archVariables = common.MapStr{
	// Regular function call parameters 1 to 6
	"P1": "%di",
	"P2": "%si",
	"P3": "%dx",
	"P4": "%cx",
	"P5": "%r8",
	"P6": "%r9",

	// System call parameters. These are temporary, the definitive SYS_Px args
	// will be determined by guess/syscallargs.go.
	"_SYS_P1": "%di",
	"_SYS_P2": "%si",
	"_SYS_P3": "%dx",
	"_SYS_P4": "%cx", // This already translated from r10 by syscall handler
	"_SYS_P5": "%r8",
	"_SYS_P6": "%r9",

	"RET": "%ax",
}
