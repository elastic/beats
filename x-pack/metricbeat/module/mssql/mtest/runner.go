// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package mtest

import (
	"github.com/elastic/beats/libbeat/tests/compose"
)

var (
	// Runner is a compose test runner for Redis tests
	Runner = compose.TestRunner{
		Service: "mssql",
	}
)
