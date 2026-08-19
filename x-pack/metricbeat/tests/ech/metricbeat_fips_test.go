// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build ech && requirefips

package fips

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v9/testing/go-ech"
)

const metricbeatFIPSConfig = `
metricbeat.modules:
  - module: system
    enabled: true
    period: 5s
    metricsets:
      - cpu
      - memory
path.home: %s
logging.to_files: false
logging.to_stderr: true
output:
  elasticsearch:
    hosts:
      - ${ES_HOST}
    protocol: https
    username: ${ES_USER}
    password: ${ES_PASS}
`

// TestMetricbeatFIPSSmoke starts a FIPS compatible binary and ensures that data ends up in an (https) ES cluster.
func TestMetricbeatFIPSSmoke(t *testing.T) {
	ech.VerifyEnvVars(t)
	ech.VerifyFIPSBinary(t, "../../metricbeat")

	ech.RunSmokeTest(t,
		"metricbeat",
		"../../metricbeat",
		fmt.Sprintf(metricbeatFIPSConfig, t.TempDir()),
	)
}
