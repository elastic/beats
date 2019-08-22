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
	"P4":  "+4(%sp)",
	"P5":  "+8(%sp)",
	"P6":  "+12(%sp)",
	"RET": "%ax",

	// System call parameters
	"SYS_P1": "+4(%sp)",
	"SYS_P2": "+8(%sp)",
	"SYS_P3": "+12(%sp)",
	"SYS_P4": "+16(%sp)",
	"SYS_P5": "+20(%sp)",
	"SYS_P6": "+24(%sp)",
}
