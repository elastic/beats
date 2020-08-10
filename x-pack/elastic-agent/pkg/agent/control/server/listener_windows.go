// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package server

import (
	"net"
	"os/user"

	"github.com/elastic/beats/v7/libbeat/api/npipe"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// createListener creates a named pipe listener on Windows
func createListener(_ *logger.Logger) (net.Listener, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	sd, err := npipe.DefaultSD(u.Username)
	if err != nil {
		return nil, err
	}
	return npipe.NewListener(control.Address(), sd)
}

func cleanupListener(_ *logger.Logger) {
	// nothing to do on windows
}
