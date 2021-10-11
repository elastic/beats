// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package control

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// Address returns the address to connect to Elastic Agent daemon.
func Address() string {
	// when installed the control address is fixed
	if info.RunningInstalled() {
		return paths.SocketPath
	}

	// unix socket path must be less than 104 characters
	path := fmt.Sprintf("unix://%s.sock", filepath.Join(paths.TempDir(), "elastic-agent-control"))
	if len(path) < 104 {
		return path
	}
	// place in global /tmp to ensure that its small enough to fit; current path is way to long
	// for it to be used, but needs to be unique per Agent (in the case that multiple are running)
	return fmt.Sprintf(`unix:///tmp/elastic-agent/%x.sock`, sha256.Sum256([]byte(path)))
}
