// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

// Package npcap provides an embedded Npcap OEM installer. The embedded installer
// must be placed in the installer directory and have a name that matches the pattern
// "npcap-([0-9]\.[0-9]+)(?:|-oem)\.exe" where the capture is the installer version.
package npcap

import (
	"embed"
	"fmt"
	"path"
	"strings"

	"github.com/elastic/beats/v7/packetbeat/npcap"
)

//go:embed installer/*.exe
var fs embed.FS

func init() {
	list, err := fs.ReadDir("installer")
	if err != nil {
		panic(fmt.Sprintf("failed to set up npcap installer: %v", err))
	}
	if len(list) == 0 {
		return
	}
	if len(list) > 1 {
		panic(fmt.Sprintf("unexpected number of installers found: want only one but got %d", len(list)))
	}
	installer := list[0].Name()

	version := strings.TrimPrefix(installer, "npcap-")
	version = strings.TrimSuffix(version, ".exe")
	version = strings.TrimSuffix(version, "-oem")
	npcap.EmbeddedInstallerVersion = version

	npcap.Installer, err = fs.ReadFile(path.Join("installer", installer))
	if err != nil {
		panic(fmt.Sprintf("failed to set up npcap installer: %v", err))
	}
}
