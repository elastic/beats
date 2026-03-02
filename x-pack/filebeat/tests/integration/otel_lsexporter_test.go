// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type event struct {
	ID           string                 `json:"id"`
	Timestamp    string                 `json:"timestamp"`
	StringField  string                 `json:"string_field"`
	NumberField  int                    `json:"number_field"`
	FloatField   float64                `json:"float_field"`
	BooleanField bool                   `json:"boolean_field"`
	ArrayField   []interface{}          `json:"array_field"`
	ObjectField  map[string]interface{} `json:"object_field"`
	KVField      map[string]interface{} `json:"kv_field"`
}

type eventWithID struct {
	id   string
	data mapstr.M
}

// TestDataShapeOTelVSClassicE2E verifies that the event data shape of filebeat in otel mode is the same as filebeat.
// Two Filebeat instances are started:
//
//	one for classic mode sending to Logstash on port 5044 and
//	one for Otel mode sending to Logstash on port 5055.
//
// Logstash runs two pipelines listening on those ports and writes the resulting events into a shared Docker volume.
// Nginx container serves that volume over HTTP so the test can fetch the generated files without relying on host filesystem permissions.
// Finally, the test downloads both files to ./tests/integration/logstash/testdata and compares the sorted events line by line.
func TestDataShapeOTelVSClassicE2E(t *testing.T) {
	// ensure the size of events is big enough (1024b) for filebeat to ingest
	numEvents := 3

	// Agent does not support `index` setting, while beats does.
	//	Our focus is on agent classic vs otel mode comparison, so we do not test `index` for filebeat
	beatsCfgFile := `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    paths:
      - %s
output.logstash:
  hosts:
    - "localhost:%s"
  pipelining: 0
  worker: 1
queue.mem.flush.timeout: 0s
processors:
  - add_host_metadata: ~
  - add_fields:
      target: ""
      fields:
        testcase: %s
`
	testCaseName := uuid.Must(uuid.NewV4()).String()
	events := generateEvents(numEvents)

	tmpdir := t.TempDir()
	inputFilePath := filepath.Join(tmpdir, "event.json")
	writeEvents(t, inputFilePath, events)

	otelConfig := fmt.Sprintf(`receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    processors:
      - add_host_metadata: ~
      - add_fields:
          target: ""
          fields:
            testcase: %s
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
exporters:
  logstash:
    hosts:
      - "localhost:5055"
    tls:
      insecure: true
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - logstash
  telemetry:
    metrics:
      level: none
`, inputFilePath, testCaseName, tmpdir)

	// Start OTel collector with filebeatreceiver
	oteltestcol.New(t, otelConfig)

	// start filebeat
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	inputFilePath = filepath.Join(filebeat.TempDir(), "event.json")
	writeEvents(t, inputFilePath, events)

	filebeat.WriteConfigFile(fmt.Sprintf(beatsCfgFile, inputFilePath, "5044", testCaseName))
	filebeat.Start()
	defer filebeat.Stop()

	// Nginx endpoint URLs
	baseURL := "http://localhost:8082"
	outFileURL := fmt.Sprintf("%s/%s_fb.json", baseURL, testCaseName)
	outOTelFileURL := fmt.Sprintf("%s/%s_otel.json", baseURL, testCaseName)

	// wait for logs to be published over HTTP
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			for _, url := range []string{outFileURL, outOTelFileURL} {
				checkURLHasContent(ct, url)
			}
		},
		30*time.Second, 1*time.Second, "expected Nginx to serve json files over HTTP")

	// download files from Nginx into testdata directory
	fbFilePath := downloadToTestData(t, outFileURL, fmt.Sprintf("%s_fb.json", testCaseName))
	otelFilePath := downloadToTestData(t, outOTelFileURL, fmt.Sprintf("%s_otel.json", testCaseName))

	ignoredFields := []string{
		// Expected to change between agentDocs and OtelDocs
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"log.file.inode",
		"log.file.path",
		// only present in beats receivers
		"log.file.device_id",
		"log.file.fingerprint",
	}

	compareOutputFiles(t, fbFilePath, otelFilePath, ignoredFields)
}

// TestLogstashExporterProxyURL verifies that Filebeat OTel mode can send data to Logstash via a SOCKS5 proxy.
// Filebeat otel mode sends events to "logstash" via a socks5-proxy container running on localhost:1080
func TestLogstashExporterProxyURL(t *testing.T) {
	// ensure the size of events is big enough
	numEvents := 3

	testCaseName := uuid.Must(uuid.NewV4()).String()
	events := generateEvents(numEvents)

	// Create OTel collector configuration with filebeatreceiver
	tmpdir := t.TempDir()
	inputFilePath := filepath.Join(tmpdir, "event.json")
	writeEvents(t, inputFilePath, events)

	otelConfig := fmt.Sprintf(`receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: filestream-input-id
          enabled: true
          paths:
            - %s
          prospector.scanner.fingerprint.enabled: false
          file_identity.native: ~
    processors:
      - add_host_metadata: ~
      - add_fields:
          target: ""
          fields:
            testcase: %s
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
exporters:
  logstash:
    hosts:
      - "logstash:5055"
    ttl: 0s
    proxy_url: "socks5://elastic:changeme@localhost:1080"
    proxy_use_local_resolver: false
    worker: 1
    workers: 0
    max_retries: 3
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - logstash
  telemetry:
    metrics:
      level: none
`, inputFilePath, testCaseName, tmpdir)

	// Start OTel collector with filebeatreceiver
	oteltestcol.New(t, otelConfig)

	// Nginx endpoint URLs
	baseURL := "http://localhost:8082"
	outOTelFileURL := fmt.Sprintf("%s/%s_otel.json", baseURL, testCaseName)

	// Logstash is outputting to a file inside its container, to access
	// this file we use Nginx to serve the output folder via HTTP
	// (see docker-compose.yml for Logstash and Nginx configuration).
	// Wait to ensure the file can be downloaded via HTTP, the file
	// being available indicates that Filebeat successfully sent data
	// to Logstash
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			for _, url := range []string{outOTelFileURL} {
				checkURLHasContent(ct, url)
			}
		},
		30*time.Second,
		1*time.Second,
		"did not find Logstash output file served via Nginx")

	// download files from Nginx
	otelFilePath := downloadToTestData(t, outOTelFileURL, fmt.Sprintf("%s_otel.json", testCaseName))
	otelEvents, err := readAllEvents(otelFilePath)

	require.NoError(t, err, "failed to read otel events")
	require.Len(t, otelEvents, numEvents,
		"different number of events: sent=%d, get=%d", numEvents, len(otelEvents))
}

func generateEvents(numEvents int) []string {
	gofakeit.Seed(time.Now().UnixNano())

	events := make([]string, 0, numEvents)
	for i := 0; i < numEvents; i++ {
		// Generate mixed-type array field
		arrayField := make([]interface{}, 9)
		arrayField[0] = gofakeit.Word()
		arrayField[1] = gofakeit.Int64()
		arrayField[2] = gofakeit.Float64()
		arrayField[3] = rand.IntN(2) == 0 // bool
		arrayField[4] = gofakeit.Name()
		arrayField[5] = math.MaxInt
		arrayField[6] = math.MinInt
		arrayField[7] = math.MaxFloat64
		arrayField[8] = math.SmallestNonzeroFloat64

		kvArrayField := make([]interface{}, 4)
		kvArrayField[0] = gofakeit.Color()
		kvArrayField[1] = gofakeit.Number(-100, 100)
		kvArrayField[2] = gofakeit.Float32Range(0, 50)
		kvArrayField[3] = rand.IntN(2) == 0 // bool

		ev := event{
			ID:           uuid.Must(uuid.NewV4()).String(),
			Timestamp:    time.Now().Format(time.RFC3339Nano),
			StringField:  gofakeit.Sentence(2),
			NumberField:  rand.IntN(1000),
			FloatField:   rand.Float64() * 100,
			BooleanField: rand.IntN(2) == 0,
			ArrayField:   arrayField,
			ObjectField: map[string]interface{}{
				"nested_key":    "nested_value",
				"nested_number": gofakeit.Number(1, 1000),
			},
			KVField: map[string]interface{}{
				"key_string": gofakeit.Word(),
				"key_number": gofakeit.Number(1, 5000),
				"key_bool":   rand.IntN(2) == 0,
				"key_array":  kvArrayField,
				"key_object": map[string]interface{}{
					"inner1": rand.IntN(2) == 0,
					"inner2": gofakeit.Float64Range(0, 10),
					"inner_obj": map[string]interface{}{
						"deep_key": gofakeit.HipsterSentence(3),
						"deep_arr": kvArrayField,
					},
				},
			},
		}

		b, _ := json.Marshal(ev)
		events = append(events, string(b))
	}
	return events
}

func writeEvents(t *testing.T, filepath string, events []string) {
	f, err := os.Create(filepath)
	if err != nil {
		t.Fatalf("cannot create file '%s': %s", filepath, err)
	}

	for _, event := range events {
		if _, err := f.WriteString(event + "\n"); err != nil {
			t.Fatalf("cannot write log file '%s': %s", filepath, err)
		}
	}

	if err := f.Sync(); err != nil {
		t.Errorf("cannot sync %q: %s", filepath, err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("cannot close %q: %s", filepath, err)
	}
}

func parseJson(jsonStr string) (mapstr.M, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}
	return data, nil
}

func compareOutputFiles(t *testing.T, fbFilePath, otelFilePath string, ignoredFields []string) {
	fbEvents, err := readAllEvents(fbFilePath)
	require.NoError(t, err, "failed to read filebeat events")

	otelEvents, err := readAllEvents(otelFilePath)
	require.NoError(t, err, "failed to read otel events")

	require.Len(t, otelEvents, len(fbEvents),
		"different number of events: filebeat=%d, otel=%d", len(fbEvents), len(otelEvents))

	sortEventsByID(fbEvents)
	sortEventsByID(otelEvents)

	// compare sorted events
	for i := 0; i < len(fbEvents); i++ {
		fbEvent := fbEvents[i]
		otelEvent := otelEvents[i]

		oteltest.AssertMapsEqual(t, fbEvent.data, otelEvent.data, ignoredFields,
			fmt.Sprintf("event comparison failed for ID %s (line %d)", fbEvent.id, i))

		assert.Equal(t, "filebeat", otelEvent.data.Flatten()["agent.type"], "expected agent.type to be 'filebeat' in otel data")
		assert.Equal(t, "filebeat", fbEvent.data.Flatten()["agent.type"], "expected agent.type to be 'filebeat' in filebeat data")
	}
}

func readAllEvents(filePath string) ([]eventWithID, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []eventWithID
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// parse json line
		outerData, err := parseJson(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse outer JSON at line %d: %w", lineNumber, err)
		}

		// extract the message field
		messageField, exists := outerData["message"]
		if !exists {
			return nil, fmt.Errorf("missing 'message' field at line %d", lineNumber)
		}

		messageStr, ok := messageField.(string)
		if !ok {
			return nil, fmt.Errorf("'message' field is not a string at line %d", lineNumber)
		}

		// parse original event
		innerData, err := parseJson(messageStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse inner JSON from message field at line %d: %w", lineNumber, err)
		}

		// extract original event id
		id, _ := innerData["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("missing or invalid ID field in inner JSON at line %d", lineNumber)
		}

		events = append(events, eventWithID{
			id:   id,
			data: outerData,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func sortEventsByID(events []eventWithID) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].id < events[j].id
	})
}

func checkURLHasContent(ct *assert.CollectT, url string) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if !assert.NoError(ct, err, "failed to create request for URL %s", url) {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(ct, err, "URL %s should exist", url) {
		return
	}
	defer resp.Body.Close()

	if !assert.Equal(ct, http.StatusOK, resp.StatusCode, "URL %s should return HTTP 200", url) {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if !assert.NoError(ct, err, "failed to read body from %s", url) {
		return
	}

	if !assert.NotEmpty(ct, body, "URL %s should have content", url) {
		return
	}
}

func downloadToTestData(t *testing.T, url string, filename string) string {
	// get http response
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err, "error creating request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "calling nginx endpoint")
	defer resp.Body.Close()

	// get path to current file
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file path")

	// create testdata directory
	filePath := filepath.Join(filepath.Dir(currentFile), "logstash", "testdata", filename)
	err = os.MkdirAll(filepath.Dir(filePath), 0o755)
	require.NoError(t, err, "failed to create testdata directory")

	// create file
	file, err := os.Create(filePath)
	require.NoError(t, err, "failed to create file %s", filePath)
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	require.NoError(t, err, "failed to copy data from %s", url)

	err = file.Sync()
	require.NoError(t, err, "failed to sync file %s", filePath)

	return filePath
}
