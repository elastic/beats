// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && ech && requirefips

package fips

import (
	"fmt"
	"path"
	"testing"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
	"github.com/elastic/beats/v9/testing/go-ech"
)

const filebeatFIPSConfig = `
filebeat.inputs:
  - type: filestream
    id: "test-filebeat-fips"
    paths:
      - %s
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

// TestFilebeatFIPSSmoke starts a FIPS compatible binary and ensures that data ends up in an (https) ES cluster.
func TestFilebeatFIPSSmoke(t *testing.T) {
	ech.VerifyEnvVars(t)
	ech.VerifyFIPSBinary(t, "../../filebeat")

	// Generate logs
	tempDir := t.TempDir()
	logFilePath := path.Join(tempDir, "log.log")
	integration.WriteLogFile(t, logFilePath, 1000, false)

	ech.RunSmokeTest(t,
		"filebeat",
		"../../filebeat",
		fmt.Sprintf(filebeatFIPSConfig, logFilePath, tempDir),
	)
}
