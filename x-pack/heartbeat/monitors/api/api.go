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
	"os"
	"syscall"

	"github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/security"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
)

func init() {
	// Reuse browser's params-insensitive config hash so Fleet parameter pushes
	// don't trigger unnecessary monitor stop/restarts.
	plugin.RegisterWithHashFunc("api", browser.HashConfig, create, "synthetics/api")
}

func create(name string, cfg *config.C, _ beat.Info) (p plugin.Plugin, err error) {
	// API journeys don't launch Chromium, but they run the same Node.js
	// synthetics agent as browser monitors, which is only present where
	// ELASTIC_SYNTHETICS_CAPABLE is set. Gate on it like browser does.
	if os.Getenv("ELASTIC_SYNTHETICS_CAPABLE") != "true" {
		return plugin.Plugin{}, ecserr.NewNotSyntheticsCapableError()
	}

	// We do not use user.Current() which does not reflect setuid changes!
	if syscall.Geteuid() == 0 && security.NodeChildProcCred == nil {
		return plugin.Plugin{}, fmt.Errorf("api monitors cannot be run as root")
	}

	sj, err := browser.NewSourceJob(cfg)
	if err != nil {
		return plugin.Plugin{}, err
	}

	return sj.Plugin(), nil
}
