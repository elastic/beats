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

package ech

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// VerifyEnvVars ensures that the env vars to connect to ES are set, and that the ES_HOST starts with https
func VerifyEnvVars(t *testing.T) {
	t.Helper()
	esHost := os.Getenv("ES_HOST")
	assert.NotEmpty(t, esHost, "Expected env var ES_HOST to be not-empty.")
	assert.Regexp(t, regexp.MustCompile(`^https://`), esHost)
	esUser := os.Getenv("ES_USER")
	assert.NotEmpty(t, esUser, "Expected env var ES_USER to be not-empty.")
	esPass := os.Getenv("ES_PASS")
	assert.NotEmpty(t, esPass, "Expected env var ES_PASS to be not-empty.")
	if t.Failed() {
		t.Fatal("Missing expected env var.") // stop test if an assertion fails
	}
}

// VerifyFIPSBinary ensures the binary on binaryPath has FIPS indicators.
func VerifyFIPSBinary(t *testing.T, binaryPath string) {
	t.Helper()
	info, err := buildinfo.ReadFile(binaryPath)
	assert.NoError(t, err)

	var checkLinks, foundTags, foundExperiment bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "-tags":
			foundTags = true
			assert.Contains(t, setting.Value, "requirefips")
			assert.Contains(t, setting.Value, "ms_tls13kdf")
			continue
		case "GOEXPERIMENT":
			foundExperiment = true
			assert.Contains(t, setting.Value, "systemcrypto")
			continue
		case "-ldflags":
			if !strings.Contains(setting.Value, "-s") {
				checkLinks = true
			}
		}
	}

	assert.True(t, foundTags, "did not find build tags")
	assert.True(t, foundExperiment, "did not find GOEXPERIMENT")

	if checkLinks && runtime.GOOS == "linux" {
		t.Log("Binary is not stripped, checking for OpenSSL in the symbols table.")
		output, err := exec.CommandContext(t.Context(), "go", "tool", "nm", binaryPath).Output()
		assert.NoError(t, err, "unable to run go tool nm")
		assert.Contains(t, output, "OpenSSL_version", "Unable to find OpenSSL_version in symbols link")
	}
	if t.Failed() {
		t.Fatal("Unable to verify FIPS binary.") // stop test if non-FIPS binary is used.
	}
}

// RunSmokeTest runs the beat on binaryPath with the passed config, and ensures that data ends up in Elasticsearch.
func RunSmokeTest(t *testing.T, name, binaryPath, cfg string) {
	proc := integration.NewStandardBeat(t, name, binaryPath)
	proc.WriteConfigFile(cfg)

	// start binary
	proc.Start()
	defer proc.Stop()

	// ensure data ends up in ES
	es, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{os.Getenv("ES_HOST")},
		Username:  os.Getenv("ES_USER"),
		Password:  os.Getenv("ES_PASS"),
	})
	require.NoError(t, err, "unable to create elasticsearch client")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := es.Search().Index(name + "-*").Do(t.Context())
		require.NoError(c, err, "search request for index failed.")
		require.NotZero(c, resp.Hits.Total.Value, "expected to find hits within ES.")
	}, time.Minute, time.Second, name+" documents are not detected within the elasticsearch deployment")
}
