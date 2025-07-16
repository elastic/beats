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

//go:build integration

package fips

import (
	"fmt"
	"os"
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
	esUser := os.Getenc("ES_USER")
	require.NotEmpty(t, esUser, "Expected env var ES_USER to be not-empty.")
	esPass := os.Getenc("ES_PASS")
	require.NotEmpty(t, esPass, "Expected env var ES_PASS to be not-empty.")

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat",
	)
	filebeat.RemoveAllCLIArgs()

	// 1. Generate the log file path, but do not write data to it
	tempDir := filebeat.TempDir()
	logFilePath := path.Join(tempDir, "log.log")

	// 2. Create the log file
	integration.GenerateLogFile(t, logFilePath, 10, false)

	// 3. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filebeatFIPSConfig, logFilePath, tempDir, esHost, esUser, esPass))
	filebeat.Start()

	es, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{esHost},
		Username:  esUser,
		Password:  esPass,
	})
	require.NoError(t, err, "unable to create elasticsearch client")

	// 4. Ensure data ends up in ES
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := es.Search().Index("logs-*").Do(t.Context())
		require.NoError(c, err, "search request for index failed.")
		require.Equal(t, int64(10), resp.Hits.Total.Value, "expected to find 10 hits within ES.")
	}, time.Minute, time.Second, "filebeat logs are not detected within the elasticsearch deployment")
}
