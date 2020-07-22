// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package server

import (
	"net"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
)

func createListener() (net.Listener, error) {
	return net.Listen("unix", control.Address())
}
