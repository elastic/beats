// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/tests/integration"
	"github.com/elastic/beats/v9/libbeat/version"
)

func TestIndexTotalFieldsLimitNotReached(t *testing.T) {
	cfg := `
metricbeat:
logging:
  level: debug
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
  reload.enabled: false
`
	metricbeat := integration.NewBeat(t, "metricbeat", "../../metricbeat.test")
	metricbeat.WriteConfigFile(cfg)
	esURL := integration.GetESURL(t, "http")
	kURL, _ := integration.GetKibana(t)

	ver, _, _ := strings.Cut(version.GetDefaultVersion(), "-")
	index := "metricbeat-" + ver

	dataStreamURL, err := integration.FormatDatastreamURL(t, esURL, index)
	require.NoError(t, err)
	templateURL, err := integration.FormatIndexTemplateURL(t, esURL, index)
	require.NoError(t, err)
	policyURL, err := integration.FormatPolicyURL(t, esURL, index)
	cleanUpES := func() {
		_, _, err := integration.HttpDo(t, http.MethodDelete, dataStreamURL)
		require.NoErrorf(t, err, "cleanup failed: could not remove datastream %s", index)
		_, _, err = integration.HttpDo(t, http.MethodDelete, templateURL)
		require.NoErrorf(t, err, "cleanup failed: could not remove index template %s", index)
		_, _, err = integration.HttpDo(t, http.MethodDelete, policyURL)
		require.NoErrorf(t, err, "cleanup failed: could not remove ilm policy %s", index)
	}
	// ensure no datastream/index template/ilm policy is set before running the test
	cleanUpES()
	t.Cleanup(cleanUpES)

	metricbeat.Start("setup",
		"--index-management",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "output.elasticsearch.protocol=http",
		"-E", "output.elasticsearch.hosts=['"+esURL.String()+"']")
	err = metricbeat.Cmd.Wait()
	require.NoError(t, err, "metricbeat setup failed")
	require.Equalf(t, 0, metricbeat.Cmd.ProcessState.ExitCode(),
		"metricbeat setup failed: incorrect exit code: %d", metricbeat.Cmd.ProcessState.ExitCode())

	// generate an event with dynamically mapped fields
	fields := map[string]string{}
	totalFields := 500
	for i := range totalFields {
		fields[fmt.Sprintf("a-label-%d", i)] = fmt.Sprintf("some-value-%d", i)
	}
	event, err := json.Marshal(map[string]any{
		"@timestamp": time.Now().Format(time.RFC3339),
		// 'kubernetes.labels.*' is a dynamically mapped field
		"kubernetes.labels": fields,
	})
	require.NoError(t, err, "could not marshal event to send to ES")

	endpoint := fmt.Sprintf("%s/%s/_doc", esURL.String(), index)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(event))
	require.NoError(t, err, "could not create request to send event to ES")
	r.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(r)
	require.NoError(t, err, "could not send request to send event to ES")
	defer resp.Body.Close()

	failuremsg := fmt.Sprintf("failed to ingest events with %d new fields. If this test fails it likely means the current `index.mapping.total_fields.limit` for metricbeat index (%s) is close to be reached. Check the logs to see why the event was not ingested", totalFields, index)
	if !assert.Equal(t, http.StatusCreated, resp.StatusCode, failuremsg) {
		t.Logf("event sent: %s", string(event))

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "could not read response body")
		t.Logf("ES ingest event reponse: %s", string(respBody))
	}
}
