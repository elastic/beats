// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat
// +build integration,!agentbeat

package system

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func runPacketbeat(t testing.TB, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	packetbeatPath, err := filepath.Abs("../../packetbeat.test")
	require.NoError(t, err)

	if _, err := os.Stat(packetbeatPath); err != nil {
		t.Fatalf("%v binary not found: %v", filepath.Base(packetbeatPath), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conf, err := filepath.Abs("../../packetbeat.yml")
	if err != nil {
		return "", "", err
	}
	cmd := exec.CommandContext(ctx, packetbeatPath, append([]string{"-systemTest", "-c", conf}, args...)...)
	cmd.Dir = t.TempDir()
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()

	return strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()), err
}
