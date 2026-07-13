// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

// Package api registers the `api` monitor type: synthetics API journeys that
// run the Node.js agent like `browser` but never launch Chromium. This file is
// a thin registration shim over the sibling `browser` package.
package api

import (
	"fmt"
	"syscall"

	"github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/security"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
)

func init() {
	plugin.Register("api", create, "synthetics/api")
}

func create(name string, cfg *config.C, _ beat.Info) (p plugin.Plugin, err error) {
	// API journeys still spawn a Node.js child process (setuid guard applies) but
	// need no GUI libs, so we skip browser's ELASTIC_SYNTHETICS_CAPABLE gate.
	if syscall.Geteuid() == 0 && security.NodeChildProcCred == nil {
		return plugin.Plugin{}, fmt.Errorf("api monitors cannot be run as root")
	}

	sj, err := browser.NewSourceJob(cfg)
	if err != nil {
		return plugin.Plugin{}, err
	}

	return sj.Plugin(), nil
}
