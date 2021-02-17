// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package control

import (
	"crypto/sha256"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
)

// Address returns the address to connect to Elastic Agent daemon.
func Address() string {
	// when installed the control address is fixed
	if install.RunningInstalled() {
		return install.SocketPath
	}

	// not install, adjust the path based on data path
	data := paths.Data()
	// entire string cannot be longer than 256 characters, this forces the
	// length to always be 87 characters (but unique per data path)
	return fmt.Sprintf(`\\.\pipe\elastic-agent-%x`, sha256.Sum256([]byte(data)))
}
