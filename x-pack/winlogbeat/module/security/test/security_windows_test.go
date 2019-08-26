// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"testing"

	"github.com/elastic/beats/x-pack/winlogbeat/module"
)

func TestSecurity(t *testing.T) {
	module.TestPipeline(t, "testdata/*.evtx", "../config/winlogbeat-security.js")
}
