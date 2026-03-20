// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestMain(m *testing.M) {
	binPath, err := filepath.Abs("../../filebeat.test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve binary path: %s\n", err)
		os.Exit(1)
	}
	packagePath, err := filepath.Abs("../../")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve package path: %s\n", err)
		os.Exit(1)
	}
	if err := integration.BuildSystemTestBinary(binPath, packagePath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build filebeat test binary: %s\n", err)
		os.Exit(1)
	}

	rc := m.Run()

	_ = os.Remove(binPath)
	os.Exit(rc)
}
