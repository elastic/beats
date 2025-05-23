// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
)

// Test all required plugins are exported by this module, since it's the
// one imported by agentbeat: https://github.com/elastic/beats/pull/39818
func TestRootCmdPlugins(t *testing.T) {
	t.Parallel()
	plugins := []string{"http", "tcp", "icmp", "browser"}
	for _, p := range plugins {
		t.Run(fmt.Sprintf("%s plugin", p), func(t *testing.T) {
			_, found := plugin.GlobalPluginsReg.Get(p)
			assert.True(t, found)
		})
	}
}
