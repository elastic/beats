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
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/packetbeat/npcap"
)

// Keep in sync with NpcapVersion in magefile.go.
const NpcapVersion = "1.79"

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
