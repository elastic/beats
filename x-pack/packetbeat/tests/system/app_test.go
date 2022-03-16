// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package system

import (
	"bytes"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/packetbeat/npcap"
)

func TestWindowsNpcapInstaller(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("skipping non-Windows GOOS: %s", runtime.GOOS)
	}

	stdout, stderr, err := runPacketbeat(t, "devices")
	require.NoError(t, err, stderr)
	t.Log("Output:\n", stdout)

	b, err := os.ReadFile(`C:\Program Files\Npcap\install.log`)
	if err != nil {
		return fmt.Errorf("could not read install.log: %w", err)
	}
	// From inspection we expect a line "DetailPrint: Starting the npcap driver".
	if !bytes.Contains(b, []byte("Starting the npcap driver")) {
		return errors.New("install log does not include npcap drives start line")
	}

	installedNpcapVersion := npcap.Version()
	if !strings.Contains(installedNpcapVersion, NpcapVersion) {
		return fmt.Errorf("unexpected npcap version installed: want:%s have:%s", NpcapVersion, installedNpcapVersion)
	}
}

func TestDevices(t *testing.T) {
	stdout, stderr, err := runPacketbeat(t, "devices")
	require.NoError(t, err, stderr)
	t.Log("Output:\n", stdout)

	ifcs, err := net.Interfaces()
	require.NoError(t, err)

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

	cmd := exec.Command(packetbeatPath, append([]string{"-systemTest"}, args...)...)
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
