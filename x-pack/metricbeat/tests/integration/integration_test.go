// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/dev-tools/testbin"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestMain(m *testing.M) {
	var opts []testbin.Option
	// On Windows 7 32-bit we run out of memory if we enable DWARF.
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		opts = append(opts, testbin.WithExtraFlags("-ldflags=-w"))
	}
	integration.TestMainWithBuild(m, "metricbeat", opts...)
}
