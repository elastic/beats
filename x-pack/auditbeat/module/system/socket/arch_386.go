// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import "github.com/elastic/beats/libbeat/common"

var archVariables = common.MapStr{
	// Regular function call parameters 1 to 6
	// This calling convention is used internally by the kernel
	// which is built by default with (-mregparam=3)
	"P1":  "%ax",
	"P2":  "%dx",
	"P3":  "%cx",
	"P4":  "$stack1",
	"P5":  "$stack2",
	"P6":  "$stack3",
	"RET": "%ax",

	// System call parameters
	"SYS_P1": "$stack1",
	"SYS_P2": "$stack2",
	"SYS_P3": "$stack3",
	"SYS_P4": "$stack4",
	"SYS_P5": "$stack5",
	"SYS_P6": "$stack6",
}
