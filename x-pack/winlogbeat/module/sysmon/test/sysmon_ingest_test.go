// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Windows is excluded not because the tests won't pass on Windows in general,
// but because they won't pass on Windows in a VM — where we are using this — due
// to the VM inception problem.
//
//go:build !windows

package test

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/winlogbeat/module"
)

// Ignore these fields so that the tests will pass if Sysmon is not installed.
var ignoreFields = []string{
	"event.action",
	"event.ingested",
	"message",
	"winlog.opcode",
	"winlog.task",

	// Ignore these fields as under some circumstances they are not populated.
	// (observed under Windows 7).
	"winlog.user.type",
	"winlog.user.name",
	"winlog.user.domain",
}

func TestSysmonIngest(t *testing.T) {
	module.TestIngestPipeline(t, "sysmon", "testdata/collection/*.evtx.golden.json", module.WithFieldFilter(ignoreFields))
}
