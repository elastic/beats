// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin
// +build linux darwin

package browser

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/security"
)

func init() {
	plugin.Register("browser", create, "synthetic", "synthetics/synthetic")
}

var showExperimentalOnce = sync.Once{}

func create(name string, cfg *config.C) (p plugin.Plugin, err error) {
	// We don't want users running synthetics in environments that don't have the required GUI libraries etc, so we check
	// this flag. When we're ready to support the many possible configurations of systems outside the docker environment
	// we can remove this check.
	if os.Getenv("ELASTIC_SYNTHETICS_CAPABLE") != "true" {
		return plugin.Plugin{}, ecserr.NewNotSyntheticsCapableError()
	}

	showExperimentalOnce.Do(func() {
		logp.Info("Synthetic browser monitor detected! Please note synthetic monitors are a beta feature!")
	})

	// We do not use user.Current() which does not reflect setuid changes!
	if syscall.Geteuid() == 0 && security.NodeChildProcCred == nil {
		return plugin.Plugin{}, fmt.Errorf("script monitors cannot be run as root")
	}

	s, err := NewProject(cfg)
	if err != nil {
		return plugin.Plugin{}, err
	}

	return s.plugin(), nil
}
