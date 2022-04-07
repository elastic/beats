// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package socket

import "github.com/elastic/beats/v8/libbeat/common"

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
	"_SYS_P1": "$stack1",
	"_SYS_P2": "$stack2",
	"_SYS_P3": "$stack3",
	"_SYS_P4": "$stack4",
	"_SYS_P5": "$stack5",
	"_SYS_P6": "$stack6",
}
