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
// +build integration

package filestream

import (
	"context"
	"testing"
)

func TestParsersAgentLogs(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"ndjson": map[string]interface{}{
					"message_key":    "log",
					"overwrite_keys": true,
				},
			},
		},
	})

	testline := []byte("{\"log.level\":\"info\",\"@timestamp\":\"2021-05-12T16:15:09.411+0000\",\"log.origin\":{\"file.name\":\"log/harvester.go\",\"file.line\":302},\"message\":\"Harvester started for file: /var/log/auth.log\",\"ecs.version\":\"1.6.0\"}\n")
	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testline))

	env.requireEventContents(0, "message", "Harvester started for file: /var/log/auth.log")
	env.requireEventContents(0, "log.level", "info")
	env.requireEventTimestamp(0, "2021-05-12T16:15:09.411")

	cancelInput()
	env.waitUntilInputStops()
}

// test_docker_logs_filtering from test_json.py
func TestParsersDockerLogsFiltering(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"ndjson": map[string]interface{}{
					"message_key": "log",
					"target":      "",
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
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testline))

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
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"ndjson": map[string]interface{}{
					"message_key":    "message",
					"target":         "",
					"overwrite_keys": true,
				},
			},
		},
	})

	testline := []byte("{\"source\": \"hello\", \"message\": \"test source\"}\n")
	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testline))

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
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"ndjson": map[string]interface{}{
					"target":         "",
					"overwrite_keys": true,
					"add_error_key":  true,
				},
			},
		},
	})

	testline := []byte(`{"@timestamp":"2016-04-05T18:47:18.444Z", "msg":"hallo"}
{"@timestamp":"invalid"}
{"@timestamp":{"hello": "test"}}
`)

	env.mustWriteLinesToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testline))

	env.requireEventTimestamp(0, "2016-04-05T18:47:18.444")
	env.requireEventContents(1, "error.message", "@timestamp not overwritten (parse error on invalid)")
	env.requireEventContents(2, "error.message", "@timestamp not overwritten (not string)")

	cancelInput()
	env.waitUntilInputStops()
}

// test_java_elasticsearch_log from test_multiline.py
func TestParsersJavaElasticsearchLogs(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "^\\[",
					"negate":  true,
					"match":   "after",
					"timeout": "100ms", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(elasticsearchMultilineLogs)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(20)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	cancelInput()
	env.waitUntilInputStops()
}

// test_c_style_log from test_multiline.py
func TestParsersCStyleLog(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "\\\\$",
					"negate":  false,
					"match":   "before",
					"timeout": "100ms", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(`The following are log messages
This is a C style log\\
file which is on multiple\\
lines
In addition it has normal lines
The total should be 4 lines covered
`)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(4)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	cancelInput()
	env.waitUntilInputStops()
}

// test_rabbitmq_multiline_log from test_multiline.py
func TestParsersRabbitMQMultilineLog(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "^=[A-Z]+",
					"negate":  true,
					"match":   "after",
					"timeout": "3s", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(`=ERROR REPORT==== 3-Feb-2016::03:10:32 ===
connection <0.23893.109>, channel 3 - soft error:
{amqp_error,not_found,
            "no queue 'bucket-1' in vhost '/'",
            'queue.declare'}
=ERROR REPORT==== 3-Feb-2016::03:10:32 ===
connection <0.23893.109>, channel 3 - soft error:
{amqp_error,not_found,
            "no queue 'bucket-1' in vhost '/'",
            'queue.declare'}
`)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	cancelInput()
	env.waitUntilInputStops()
}

// test_max_lines from test_multiline.py
func TestParsersMultilineMaxLines(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":      "pattern",
					"pattern":   "^\\[",
					"negate":    true,
					"match":     "after",
					"max_lines": 3,
					"timeout":   "3s", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(elasticsearchMultilineLongLogs)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.requireEventsReceived([]string{
		"[2015-12-06 01:44:16,735][INFO ][node                     ] [Zach] version[2.0.0], pid[48553], build[de54438/2015-10-22T08:09:48Z]",
		`[2015-12-06 01:44:53,269][DEBUG][action.admin.indices.mapping.put] [Zach] failed to put mappings on indices [[filebeat-2015.12.06]], type [process]
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)`,
		"[2015-12-06 01:44:53,646][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] create_mapping [filesystem]",
	})

	cancelInput()
	env.waitUntilInputStops()
}

// test_timeout from test_multiline.py
func TestParsersMultilineTimeout(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":      "pattern",
					"pattern":   "^\\[",
					"negate":    true,
					"match":     "after",
					"max_lines": 3,
					"timeout":   "100ms", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(`[2015] hello world
  First Line
  Second Line
`)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	moreLines := []byte(`  This should not be third
  This should not be fourth
[2016] Hello world
  First line again
`)

	env.mustAppendLinesToFile(testlogName, moreLines)

	env.requireEventsReceived([]string{
		`[2015] hello world
  First Line
  Second Line`,
	})

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines)+len(moreLines))
	env.requireEventsReceived([]string{
		`[2015] hello world
  First Line
  Second Line`,
		`  This should not be third
  This should not be fourth`,
		`[2016] Hello world
  First line again`,
	})

	cancelInput()
	env.waitUntilInputStops()
}

// test_max_bytes from test_multiline.py
func TestParsersMultilineMaxBytes(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"message_max_bytes":                 50,
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "^\\[",
					"negate":  true,
					"match":   "after",
					"timeout": "3s", // set to lower value to speed up test
				},
			},
		},
	})

	testlines := []byte(elasticsearchMultilineLongLogs)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.requireEventsReceived([]string{
		"[2015-12-06 01:44:16,735][INFO ][node             ",
		"[2015-12-06 01:44:53,269][DEBUG][action.admin.indi",
		"[2015-12-06 01:44:53,646][INFO ][cluster.metadata ",
	})

	cancelInput()
	env.waitUntilInputStops()
}

// test_close_timeout_with_multiline from test_multiline.py
func TestParsersCloseTimeoutWithMultiline(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"close.reader.after_interval":       "1s",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "^\\[",
					"negate":  true,
					"match":   "after",
				},
			},
		},
	})

	testlines := []byte(`[2015] hello world
  First Line
  Second Line
`)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
	env.waitUntilHarvesterIsDone()

	moreLines := []byte(`  This should not be third
  This should not be fourth
[2016] Hello world
  First line again
`)

	env.mustAppendLinesToFile(testlogName, moreLines)

	env.requireEventsReceived([]string{
		`[2015] hello world
  First Line
  Second Line`,
	})

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines)+len(moreLines))
	env.requireEventsReceived([]string{
		`[2015] hello world
  First Line
  Second Line`,
		`  This should not be third
  This should not be fourth`,
		`[2016] Hello world
  First line again`,
	})

	cancelInput()
	env.waitUntilInputStops()
}

// test_consecutive_newline from test_multiline.py
func TestParsersConsecutiveNewline(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
		"close.reader.after_interval":       "1s",
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":    "pattern",
					"pattern": "^\\[",
					"negate":  true,
					"match":   "after",
					"timeout": "3s", // set to lower value to speed up test
				},
			},
		},
	})

	line1 := `[2016-09-02 19:54:23 +0000] Started 2016-09-02 19:54:23 +0000 "GET" for /gaq?path=%2FCA%2FFallbrook%2F1845-Acacia-Ln&referer=http%3A%2F%2Fwww.xxxxx.com%2FAcacia%2BLn%2BFallbrook%2BCA%2Baddresses&search_bucket=none&page_controller=v9%2Faddresses&page_action=show at 23.235.47.31
X-Forwarded-For:72.197.227.93, 23.235.47.31
Processing by GoogleAnalyticsController#index as JSON

  Parameters: {"path"=>"/CA/Fallbrook/1845-Acacia-Ln", "referer"=>"http://www.xxxx.com/Acacia+Ln+Fallbrook+CA+addresses", "search_bucket"=>"none", "page_controller"=>"v9/addresses", "page_action"=>"show"}
Completed 200 OK in 5ms (Views: 1.9ms)
`
	line2 := `[2016-09-02 19:54:23 +0000] Started 2016-09-02 19:54:23 +0000 "GET" for /health_check at xxx.xx.44.181
X-Forwarded-For:
SetAdCodeMiddleware.default_ad_code referer
SetAdCodeMiddleware.default_ad_code path /health_check
SetAdCodeMiddleware.default_ad_code route
`
	testlines := append([]byte(line1), []byte(line2)...)
	env.mustWriteLinesToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.requireEventsReceived([]string{
		line1[:len(line1)-1],
		line2[:len(line2)-1],
	})

	cancelInput()
	env.waitUntilInputStops()
}
