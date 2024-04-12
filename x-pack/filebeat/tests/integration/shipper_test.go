// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !windows

package integration

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operator"
)

func TestShipperInputOutput(t *testing.T) {
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, ok := esURL.User.Password()
	require.True(t, ok, "ES didn't have a password")
	kURL, kUserInfo := integration.GetKibana(t)
	kPassword, ok := kUserInfo.Password()
	require.True(t, ok, "Kibana didn't have a password")

	gRpcPath := filepath.Join(os.TempDir(), "grpc")

	// create file to ingest with a unique message
	inputFilePath := filepath.Join(t.TempDir(), "test.log")
	inputFile, err := os.Create(inputFilePath)
	require.NoError(t, err, "error creating input test file")
	uniqVal := make([]byte, 16)
	_, err = rand.Read(uniqVal)
	uniqMsg := fmt.Sprintf("%X", uniqVal)
	require.NoError(t, err, "error getting a unique random value")
	_, err = inputFile.Write([]byte(uniqMsg))
	require.NoError(t, err, "error writing input test file")
	_, err = inputFile.Write([]byte("\n"))
	require.NoError(t, err, "error writing new line")
	err = inputFile.Close()
	require.NoError(t, err, "error closing input test file")

	// Elasticsearch client
	esCfg := elasticsearch.Config{
		Addresses: []string{esURL.String()},
		Username:  esURL.User.Username(),
		Password:  esPassword,
	}
	es, err := elasticsearch.NewTypedClient(esCfg)
	require.NoError(t, err, "error creating new es client")

	cfg := `filebeat.inputs:
- type: filestream
  id: my-filestream-id
  paths:
    - %s
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug

queue.mem:
  events: 100
  flush.min_events: 0
processors:
- add_fields:
    target: data_stream
    fields:
      type: logs
      namespace: generic
      dataset: generic
- add_fields:
    target: host
    fields:
      name: %s
- add_fields:
    target: agent
    fields:
      type: metricbeat
`
	// check that file can be ingested normally and found in elasticsearch
	filebeat := NewFilebeat(t)
	filebeat.WriteConfigFile(fmt.Sprintf(cfg, inputFilePath, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword, uniqMsg))
	filebeat.Start()
	filebeat.WaitForLogs("Publish event: ", 10*time.Second)
	filebeat.WaitForLogs("PublishEvents: ", 10*time.Second)
	// It can take a few seconds for a doc to show up in a search
	require.Eventually(t, func() bool {
		res, err := es.Search().
			Index(".ds-filebeat-*").
			Request(&search.Request{
				Query: &types.Query{
					Match: map[string]types.MatchQuery{
						"message": {
							Query:    uniqMsg,
							Operator: &operator.And,
						},
					},
				},
			}).Do(context.Background())
		require.NoError(t, err, "error doing search request: %s", err)
		return res.Hits.Total.Value == 1
	}, 30*time.Second, 250*time.Millisecond, "never found document")

	shipperCfg := `filebeat.inputs:
- type: shipper
  server: unix://%s
  id: my-shipper-id
  data_stream:
    data_set: generic
    type: log
    namespace: generic
  streams:
    - id: stream-id
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug
queue.mem:
  events: 100
  flush.min_events: 0
`
	// start a shipper filebeat, wait until gRPC service starts
	shipper := NewFilebeat(t)
	shipper.WriteConfigFile(fmt.Sprintf(shipperCfg, gRpcPath, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword))
	shipper.Start()
	shipper.WaitForLogs("done setting up gRPC server", 30*time.Second)

	fb2shipperCfg := `filebeat.inputs:
- type: filestream
  id: my-filestream-id
  paths:
    - %s
output.shipper:
  server: unix://%s
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug
queue.mem:
  events: 100
  flush.min_events: 0
processors:
- script:
    lang: javascript
    source: >
      function process(event) {
          event.Put("@metadata.stream_id", "stream-id");
      }
- add_fields:
    target: data_stream
    fields:
      type: logs
      namespace: generic
      dataset: generic
- add_fields:
    target: host
    fields:
      name: %s
- add_fields:
    target: agent
    fields:
      type: metricbeat
`
	// start filebeat with shipper output, make doc is ingested into elasticsearch
	fb2shipper := NewFilebeat(t)
	fb2shipper.WriteConfigFile(fmt.Sprintf(fb2shipperCfg, inputFilePath, gRpcPath, kURL.Host, kUserInfo.Username(), kPassword, uniqMsg))
	fb2shipper.Start()
	fb2shipper.WaitForLogs("Publish event: ", 10*time.Second)
	fb2shipper.WaitForLogs("events to protobuf", 10*time.Second)
	require.Eventually(t, func() bool {
		res, err := es.Search().
			Index(".ds-filebeat-*").
			Request(&search.Request{
				Query: &types.Query{
					Match: map[string]types.MatchQuery{
						"message": {
							Query:    uniqMsg,
							Operator: &operator.And,
						},
					},
				},
			}).Do(context.Background())
		require.NoError(t, err, "error doing search request: %s", err)
		return res.Hits.Total.Value == 2
	}, 30*time.Second, 250*time.Millisecond, "never found 2 documents")

	res, err := es.Search().
		Index(".ds-filebeat-*").
		Request(&search.Request{
			Query: &types.Query{
				Match: map[string]types.MatchQuery{
					"message": {
						Query:    uniqMsg,
						Operator: &operator.And,
					},
				},
			},
		}).Do(context.Background())
	require.NoError(t, err, "error doing search request: %s", err)
	require.Equal(t, int64(2), res.Hits.Total.Value)
	diff, err := diffDocs(res.Hits.Hits[0].Source_,
		res.Hits.Hits[1].Source_)
	require.NoError(t, err, "error diffing docs")
	if len(diff) != 0 {
		t.Fatalf("docs differ:\n:%s\n", diff)
	}
}

func diffDocs(doc1 json.RawMessage, doc2 json.RawMessage) (string, error) {
	fieldsToDrop := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"elastic_agent.id",
	}
	var d1 map[string]interface{}
	var d2 map[string]interface{}

	if err := json.Unmarshal(doc1, &d1); err != nil {
		return "", err
	}

	if err := json.Unmarshal(doc2, &d2); err != nil {
		return "", err
	}
	f1 := mapstr.M(d1).Flatten()
	f2 := mapstr.M(d2).Flatten()

	for _, key := range fieldsToDrop {
		_ = f1.Delete(key)
		_ = f2.Delete(key)
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(f1.StringToPrint(), f2.StringToPrint(), false)

	if len(diffs) != 1 {
		return dmp.DiffPrettyText(diffs), nil
	}
	return "", nil
}
