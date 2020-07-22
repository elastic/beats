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
)

// createListener creates a named pipe listener on Windows
func createListener() (net.Listener, error) {
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
