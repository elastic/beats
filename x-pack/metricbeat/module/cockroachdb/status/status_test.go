// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// skipping tests on windows 32 bit versions, not supported
//go:build !integration && !windows && !386
// +build !integration,!windows,!386

package status

import (
	"os"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/menderesk/beats/v7/metricbeat/module/prometheus"
	_ "github.com/menderesk/beats/v7/metricbeat/module/prometheus/collector"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

func TestEventMapping(t *testing.T) {
	logp.TestingSetup()

	mbtest.TestDataFiles(t, "cockroachdb", "status")
}
