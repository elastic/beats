// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package system

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/packetbeat/npcap"
)

// Keep in sync with NpcapVersion in magefile.go.
const NpcapVersion = "1.60"

func TestWindowsNpcapInstaller(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("skipping non-Windows GOOS: %s", runtime.GOOS)
	}

	// Ignore error since what we care about is whether the install
	// succeeded and we do not want to fail out for irrelevant errors.
	stdout, stderr, _ := runPacketbeat(t)
	if stdout != "" {
		t.Log("Output:\n", stdout)
	}
	if stderr != "" {
		t.Log("Error:\n", stderr)
	}

	_, err := os.Stat(`C:\Program Files\Npcap\install.log`)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("could not stat install.log: %v", err)
	}

	installedNpcapVersion := npcap.Version()
	if !strings.Contains(installedNpcapVersion, NpcapVersion) {
		t.Errorf("unexpected npcap version installed: want:%s have:%s", NpcapVersion, installedNpcapVersion)
	}
}

func TestDevices(t *testing.T) {
	stdout, stderr, err := runPacketbeat(t, "devices")
	require.NoError(t, err, stderr)
	t.Log("Output:\n", stdout)

	ifcs, err := net.Interfaces()
	require.NoError(t, err)
	var expected []string
	for _, ifc := range ifcs {
		expected = append(expected, fmt.Sprintf("%d:%s:%s", ifc.Index, ifc.Name, ifc.Flags))
	}
	t.Log("Expect interfaces:\n", expected)

	for _, ifc := range ifcs {
		assert.Contains(t, stdout, ifc.Name)
	}
}

func runPacketbeat(t testing.TB, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	packetbeatPath, err := filepath.Abs(exe("../../packetbeat.test"))
	require.NoError(t, err)

	if _, err := os.Stat(packetbeatPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("%v binary not found", filepath.Base(packetbeatPath))
		}
		t.Fatal(err)
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

func exe(path string) string {
	if runtime.GOOS == "windows" {
		return path + ".exe"
	}
	return path
}
