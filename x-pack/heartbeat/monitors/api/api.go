// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

// Package api registers the `api` monitor type. API monitors run the
// embedded synthetics Node.js agent — same pipeline as `browser` — but
// never launch Chromium. They drive multi-step API checks via
// Playwright's `APIRequestContext`. The bulk of the implementation lives
// in the sibling `browser` package; this file is a thin registration
// shim plus an `api`-specific environment gate.
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
	// API journeys still run a Node.js child process, so the setuid
	// constraint that applies to browser monitors applies here too.
	// They do NOT require GUI libraries, so we deliberately skip the
	// `ELASTIC_SYNTHETICS_CAPABLE` env gate from the browser plugin.
	if syscall.Geteuid() == 0 && security.NodeChildProcCred == nil {
		return plugin.Plugin{}, fmt.Errorf("api monitors cannot be run as root")
	}

	sj, err := browser.NewSourceJob(cfg)
	if err != nil {
		return plugin.Plugin{}, err
	}

	return sj.Plugin(), nil
}
