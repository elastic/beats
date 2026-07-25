// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package system

import (
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

// Keep in sync with NpcapVersion in magefile.go.
const NpcapVersion = "1.87"

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

	// Importing the packetbeat/npcap package must not load wpcap.dll on its own,
	// since other beats would keep the DLL open and block Npcap upgrades on Windows.
	// See https://github.com/elastic/elastic-agent/issues/14517.
	out, err := runWpcapProbe(t)
	require.NoErrorf(t, err, "wpcap.dll must not be held by a process that only imports the capture code:\n%s", out)
}

func runWpcapProbe(t testing.TB) (output string, err error) {
	t.Helper()

	probe := filepath.Join(t.TempDir(), "wpcapprobe.exe")
	if b, err := exec.CommandContext(t.Context(), "go", "build", "-o", probe, filepath.FromSlash("testdata/wpcapprobe.go")).CombinedOutput(); err != nil {
		t.Fatalf("failed to build wpcap probe: %v\n%s", err, b)
	}

	b, err := exec.CommandContext(t.Context(), probe).CombinedOutput()
	return strings.TrimSpace(string(b)), err
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

ifcsLoop:
	for _, ifc := range ifcs {
		if strings.Contains(stdout, ifc.Name) {
			continue ifcsLoop
		}
		addrs, err := ifc.Addrs()
		assert.NoError(t, err)
		maddrs, err := ifc.MulticastAddrs()
		assert.NoError(t, err)
		addrs = append(addrs, maddrs...)
		for _, addr := range addrs {
			s := addr.String()
			// remove the network mask suffix
			if idx := strings.Index(s, "/"); idx > -1 {
				s = s[:idx]
			}
			if strings.Contains(stdout, s) {
				continue ifcsLoop
			}
		}
		t.Errorf("interface %q not found", ifc.Name)
	}
}
