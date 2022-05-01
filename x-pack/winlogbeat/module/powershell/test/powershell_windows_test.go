// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/winlogbeat/module"
	"github.com/elastic/go-sysinfo/providers/windows"

	// Register required processors.
	_ "github.com/elastic/beats/v7/libbeat/cmd/instance"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

// Ignore these fields because they can be different on different versions
// of windows.
var ignoreFields = []string{
	"message",
}

func TestPowerShell(t *testing.T) {
	// FIXME: We do not get opcode strings in the XML on Windows 2022, so ignore that
	// field there. Only apply this to that platform to avoid regressions elsewhere.
	// This means that golden values should be generated on a non-2022 version of
	// Windows to ensure that this field is properly rendered. This is checked in
	// the module.TestPipeline function.
	os, err := windows.OperatingSystem()
	if err != nil {
		t.Fatalf("failed to get operating system info: %v", err)
	}
	t.Logf("running tests on %s", os.Name)
	if strings.Contains(os.Name, "2022") {
		ignoreFields = append(ignoreFields, "winlog.opcode")
		t.Log("ignoring winlog.opcode")
	}

	module.TestPipeline(t, "testdata/*.evtx", module.WithFieldFilter(ignoreFields))
}
