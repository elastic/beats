// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/winlogbeat/module"

	// Register required processors.
	_ "github.com/elastic/beats/v7/libbeat/cmd/instance"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

// Ignore these fields because they can be different on different versions
// of windows.
var ignoreFields = []string{
	"message",
}

func TestSecurity(t *testing.T) {
	module.TestPipeline(t, "testdata/*.evtx", module.WithFieldFilter(ignoreFields))
}
