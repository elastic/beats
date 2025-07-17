// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build ech && requirefips

package fips

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/go-elasticsearch/v8"
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
      - %s
    protocol: https
    username: %s
    password: %s
`

// TestFilebeatFIPSSmoke starts a FIPS compatible binary and ensures that data ends up in an (https) ES cluster.
func TestFilebeatFIPSSmoke(t *testing.T) {
	// use vars directly instead of integration.GetESURL as the integration package makes assumptions about the ports and username/password.
	esHost := os.Getenv("ES_HOST")
	require.NotEmpty(t, esHost, "Expected env var ES_HOST to be not-empty.")
	esUser := os.Getenv("ES_USER")
	require.NotEmpty(t, esUser, "Expected env var ES_USER to be not-empty.")
	esPass := os.Getenv("ES_PASS")
	require.NotEmpty(t, esPass, "Expected env var ES_PASS to be not-empty.")

	// 1. Handle paths
	tempDir := t.TempDir()
	logFilePath := path.Join(tempDir, "log.log")
	configFilePath := path.Join(tempDir, "filebeat.yml")

	// 2. Create the log file
	integration.GenerateLogFile(t, logFilePath, 1000, false)

	// 3. Write configuration file
	err := os.WriteFile(configFilePath, []byte(fmt.Sprintf(filebeatFIPSConfig, logFilePath, tempDir, esHost, esUser, esPass)), 0o644)
	require.NoError(t, err, "unable to write filebeat.yml")

	// 4. Start filebeat, use a standard build directly instead of a .test build as we need to verify the FIPS builds.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, "../../filebeat", "-c", configFilePath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	defer func() {
		cancel()
		err := cmd.Wait()
		if t.Failed() {
			t.Logf("filebeat exited. err: %v\nstdout: %s\nstderr: %s\n", err, stdout.String(), stderr.String())
		}
	}()

	err = cmd.Start()
	require.NoError(t, err, "unable to start filebeat")

	// 5. Ensure data ends up in ES
	es, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{esHost},
		Username:  esUser,
		Password:  esPass,
	})
	require.NoError(t, err, "unable to create elasticsearch client")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := es.Search().Index("filebeat-*").Do(t.Context())
		require.NoError(c, err, "search request for index failed.")
		require.NotZero(c, resp.Hits.Total.Value, "expected to find hits within ES.")
	}, time.Minute, time.Second, "filebeat logs are not detected within the elasticsearch deployment")
}
