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

// +build integration

package filestream

import (
	"context"
	"testing"
)

// test_docker_logs from test_json.py
func TestParsersDockerLogs(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			map[string]interface{}{
				"ndjson": map[string]interface{}{
					"message_key": "log",
				},
			},
		},
	})

	testline := []byte("{\"log\":\"Fetching main repository github.com/elastic/beats...\\n\",\"stream\":\"stdout\",\"time\":\"2016-03-02T22:58:51.338462311Z\"}\n")
	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, len(testline))

	env.requireEventContents(0, "json.log", "Fetching main repository github.com/elastic/beats...")
	env.requireEventContents(0, "json.time", "2016-03-02T22:58:51.338462311Z")
	env.requireEventContents(0, "json.stream", "stdout")

	cancelInput()
	env.waitUntilInputStops()
}

// test_docker_logs_filtering from test_json.py
func TestParsersDockerLogsFiltering(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			map[string]interface{}{
				"ndjson": map[string]interface{}{
					"message_key":     "log",
					"keys_under_root": true,
				},
			},
		},
		"exclude_lines": []string{"main"},
	})

	testline := []byte(`{"log":"Fetching main repository github.com/elastic/beats...\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"Fetching dependencies...\n","stream":"stdout","time":"2016-03-02T22:59:04.609292428Z"}
{"log":"Execute /scripts/packetbeat_before_build.sh\n","stream":"stdout","time":"2016-03-02T22:59:04.617434682Z"}
`)
	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, len(testline))

	env.requireEventContents(0, "time", "2016-03-02T22:59:04.609292428Z")
	env.requireEventContents(0, "stream", "stdout")

	cancelInput()
	env.waitUntilInputStops()
}

// test_simple_json_overwrite from test_json.py
func TestParsersSimpleJSONOverwrite(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			map[string]interface{}{
				"ndjson": map[string]interface{}{
					"message_key":     "message",
					"keys_under_root": true,
					"overwrite_keys":  true,
				},
			},
		},
	})

	testline := []byte("{\"source\": \"hello\", \"message\": \"test source\"}\n")
	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, len(testline))

	env.requireEventContents(0, "source", "hello")
	env.requireEventContents(0, "message", "test source")

	cancelInput()
	env.waitUntilInputStops()
}

// test_timestamp_in_message from test_json.py
func TestParsersTimestampInJSONMessage(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			map[string]interface{}{
				"ndjson": map[string]interface{}{
					"keys_under_root": true,
					"overwrite_keys":  true,
					"add_error_key":   true,
				},
			},
		},
	})

	testline := []byte(`{"@timestamp":"2016-04-05T18:47:18.444Z"}
{"@timestamp":"invalid"}
{"@timestamp":{"hello": "test"}}
{"@timestamp":"2016-04-05T18:47:18.444+00:00"}
{"@timestamp":"2016-04-05T18:47:18+00:00"}
`)

	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(5)
	env.requireOffsetInRegistry(testlogName, len(testline))

	env.requireEventTimestamp(0, "2016-04-05 18:47:18.444 +0000 UTC")
	env.requireEventContents(1, "error.message", "@timestamp not overwritten (parse error on invalid)")
	env.requireEventContents(2, "error.message", "@timestamp not overwritten (not string)")
	env.requireEventTimestamp(3, "2016-04-05 18:47:18.444 +0000 +0000")
	env.requireEventTimestamp(4, "2016-04-05 18:47:18 +0000 +0000")

	cancelInput()
	env.waitUntilInputStops()
}
