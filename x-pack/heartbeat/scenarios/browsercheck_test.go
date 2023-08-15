// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build synthetics

package scenarios

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElasticSyntheticsRunnable(t *testing.T) {
	// this test should fail if synthetics isn't correctly setup in the current environment
	cmd := exec.Command("sh", "-c", "echo 'step(\"t\", () => { })' | elastic-synthetics --inline")
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Run()
	require.Equal(t, 0, cmd.ProcessState.ExitCode(), "command exited with bad code: %s", out.String())
}
