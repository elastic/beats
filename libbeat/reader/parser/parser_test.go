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

package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/multiline"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestParsersConfigSuffix(t *testing.T) {
	tests := map[string]struct {
		parsers        map[string]any
		expectedSuffix string
		expectedError  string
	}{
		"parsers with no suffix config": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{
							"stream": "all",
						},
					},
				},
			},
		},
		"parsers with correct suffix config": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{
							"stream": "stdout",
						},
					},
				},
			},
			expectedSuffix: "stdout",
		},
		"parsers with multiple suffix config": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{
							"stream": "stdout",
						},
					},
					map[string]any{
						"container": map[string]any{
							"stream": "stderr",
						},
					},
				},
			},
			expectedError: "only one stream selection is allowed",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.parsers)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)

			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), test.expectedError)
				return
			}
			require.Equal(t, test.expectedSuffix, c.Suffix)
		})
	}

}

func TestParsersConfigAndReading(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]any
		expectedMessages []string
		expectedError    string
	}{
		"no parser, no error": {
			lines:            "line 1\nline 2\n",
			parsers:          map[string]any{},
			expectedMessages: []string{"line 1\n", "line 2\n"},
		},
		"correct multiline parser": {
			lines: "line 1.1\nline 1.2\nline 1.3\nline 2.1\nline 2.2\nline 2.3\n",
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"multiline": map[string]any{
							"type":        "count",
							"count_lines": 3,
						},
					},
				},
			},
			expectedMessages: []string{
				"line 1.1\n\nline 1.2\n\nline 1.3\n",
				"line 2.1\n\nline 2.2\n\nline 2.3\n",
			},
		},
		"multiline docker logs parser": {
			lines: `{"log":"[log] The following are log messages\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] This one is\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" on multiple\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" lines","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] In total there should be 3 events\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
`,
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"keys_under_root": true,
							"message_key":     "log",
						},
					},
					map[string]any{
						"multiline": map[string]any{
							"match":   "after",
							"negate":  true,
							"pattern": "^\\[log\\]",
						},
					},
				},
			},
			expectedMessages: []string{
				"[log] The following are log messages\n",
				"[log] This one is\n\n on multiple\n\n lines",
				"[log] In total there should be 3 events\n",
			},
		},
		"non existent parser configuration": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"no_such_parser": nil,
					},
				},
			},
			expectedError: ErrNoSuchParser.Error(),
		},
		"invalid multiline parser configuration is caught before parser creation": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"multiline": map[string]any{
							"match": "after",
						},
					},
				},
			},
			expectedError: multiline.ErrMissingPattern.Error(),
		},
		"ndjson with syslog": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					{
						"ndjson": map[string]any{
							"keys_under_root": true,
							"message_key":     "log",
						},
					},
					{
						"syslog": map[string]any{
							"format":   "auto",
							"timezone": "Local",
						},
					},
				},
			},
			lines: `{"log": "<13>Jan 12 12:32:15 vagrant processd[123]: This is an RFC 3164 syslog message"}
{"log": "<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut=\"3\" eventSource=\"Application\" eventID=\"1011\"][examplePriority@32473 class=\"high\"] This is an RFC 5424 syslog message"}
{"log": "Not a valid message"}`,
			expectedMessages: []string{
				"This is an RFC 3164 syslog message",
				"This is an RFC 5424 syslog message",
				"Not a valid message",
			},
		},
		"multiline syslog": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					{
						"multiline": map[string]any{
							"match":        "after",
							"negate":       true,
							"pattern":      "^<\\d{1,3}>",
							"skip_newline": true, // This option is set since testReader does not strip newlines when splitting lines.
						},
					},
					{
						"syslog": map[string]any{
							"format": "rfc5424",
						},
					},
				},
			},
			lines: `<165>1 2003-08-24T05:14:15.000003-07:00 192.168.2.1 myproc 8710 - - [beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
<165>1 2003-08-24T05:14:20.000003-07:00 192.168.2.1 myproc 8710 - - [beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
<165>1 2003-08-24T05:14:30.000003-07:00 192.168.2.1 myproc 8710 - - This is some other debug message.`,
			expectedMessages: []string{
				`[beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
`,
				`[beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
`,
				`This is some other debug message.
`,
			},
		},
		"syslog multiline": {
			parsers: map[string]any{
				"parsers": []map[string]any{
					{
						"syslog": map[string]any{
							"format": "rfc5424",
						},
					},
					{
						"multiline": map[string]any{
							"match":        "after",
							"pattern":      "^\\s",
							"skip_newline": true, // This option is set since testReader does not strip newlines when splitting lines.
						},
					},
				},
			},
			lines: `<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - [beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
<165>1 2003-08-24T05:14:20.000003-07:00 192.168.2.1 myproc 8710 - - This is some other debug message.
<165>1 2003-08-24T05:14:30.000003-07:00 192.0.2.1 myproc 8710 - - [beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
<165>1 2003-08-24T05:14:30.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
<165>1 2003-08-24T05:14:30.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
<165>1 2003-08-24T05:14:30.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
<165>1 2003-08-24T05:14:30.000003-07:00 192.0.2.1 myproc 8710 - -     at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
`,
			expectedMessages: []string{
				`[beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
`,
				`This is some other debug message.
`,
				`[beat-logstash-some-name-832-2015.11.28] IndexNotFoundException[no such index]
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:566)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:133)
    at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver.concreteIndices(IndexNameExpressionResolver.java:77)
    at org.elasticsearch.action.admin.indices.delete.TransportDeleteIndexAction.checkBlock(TransportDeleteIndexAction.java:75)
`,
			},
		},
	}

	logger := logptest.NewTestingLogger(t, "")
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.parsers)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), test.expectedError)
				return
			}

			p := c.Create(testReader(test.lines), logger)

			i := 0
			msg, err := p.Next()
			for err == nil {
				require.Equal(t, test.expectedMessages[i], string(msg.Content))
				i++
				msg, err = p.Next()
			}
		})
	}
}

func TestJSONParsersWithFields(t *testing.T) {
	tests := map[string]struct {
		message         reader.Message
		config          map[string]any
		expectedMessage reader.Message
	}{
		"no postprocessor, no processing": {
			message: reader.Message{
				Content: []byte("line 1"),
			},
			config: map[string]any{},
			expectedMessage: reader.Message{
				Content: []byte("line 1"),
			},
		},
		"JSON post processor with keys_under_root": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\"}"),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target": "",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: mapstr.M{
					"key": "value",
				},
			},
		},
		"JSON post processor with dotted target key": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\"}"),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target": "kubernetes.audit",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: mapstr.M{
					"kubernetes": mapstr.M{
						"audit": mapstr.M{
							"key": "value",
						},
					},
				},
			},
		},
		"JSON post processor with non-dotted target key": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\"}"),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target": "kubernetes",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: mapstr.M{
					"kubernetes": mapstr.M{
						"key": "value",
					},
				},
			},
		},
		"JSON post processor with document ID": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\", \"my-id-field\":\"my-id\"}"),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target":      "",
							"document_id": "my-id-field",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: mapstr.M{
					"key": "value",
				},
				Meta: mapstr.M{
					"_id": "my-id",
				},
			},
		},
		"JSON post processor with overwrite keys and under root": {
			message: reader.Message{
				Content: []byte("{\"key\": \"value\"}"),
				Fields: mapstr.M{
					"key":       "another-value",
					"other-key": "other-value",
				},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target":         "",
							"overwrite_keys": true,
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: mapstr.M{
					"key":       "value",
					"other-key": "other-value",
				},
			},
		},
		"JSON post processor with type in message": {
			message: reader.Message{
				Content: []byte(`{"timestamp":"2016-04-05T18:47:18.444Z","level":"INFO","logger":"iapi.logger","thread":"JobCourier4","appInfo":{"appname":"SessionManager","appid":"Pooler","host":"demohost.mydomain.com","ip":"192.168.128.113","pid":13982},"userFields":{"ApplicationId":"PROFAPP_001","RequestTrackingId":"RetrieveTBProfileToken-6066477"},"source":"DataAccess\/FetchActiveSessionToken.process","msg":"FetchActiveSessionToken process ended", "type": "test"}`),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target":         "",
							"overwrite_keys": true,
							"add_error_key":  true,
							"message_key":    "msg",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte("FetchActiveSessionToken process ended"),
				Fields: mapstr.M{
					"appInfo": mapstr.M{
						"appname": "SessionManager",
						"appid":   "Pooler",
						"host":    "demohost.mydomain.com",
						"ip":      "192.168.128.113",
						"pid":     int64(13982),
					},
					"level":  "INFO",
					"logger": "iapi.logger",
					"userFields": mapstr.M{
						"ApplicationId":     "PROFAPP_001",
						"RequestTrackingId": "RetrieveTBProfileToken-6066477",
					},
					"msg":       "FetchActiveSessionToken process ended",
					"source":    "DataAccess/FetchActiveSessionToken.process",
					"thread":    "JobCourier4",
					"type":      "test",
					"timestamp": "2016-04-05T18:47:18.444Z",
				},
			},
		},
		"JSON post processor on invalid type in message": {
			message: reader.Message{
				Content: []byte(`{"timestamp":"2016-04-05T18:47:18.444Z","level":"INFO","logger":"iapi.logger","thread":"JobCourier4","appInfo":{"appname":"SessionManager","appid":"Pooler","host":"demohost.mydomain.com","ip":"192.168.128.113","pid":13982},"userFields":{"ApplicationId":"PROFAPP_001","RequestTrackingId":"RetrieveTBProfileToken-6066477"},"source":"DataAccess\/FetchActiveSessionToken.process","msg":"FetchActiveSessionToken process ended", "type": 5}`),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target":         "",
							"overwrite_keys": true,
							"add_error_key":  true,
							"message_key":    "msg",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte("FetchActiveSessionToken process ended"),
				Fields: mapstr.M{
					"appInfo": mapstr.M{
						"appname": "SessionManager",
						"appid":   "Pooler",
						"host":    "demohost.mydomain.com",
						"ip":      "192.168.128.113",
						"pid":     int64(13982),
					},
					"level":  "INFO",
					"logger": "iapi.logger",
					"userFields": mapstr.M{
						"ApplicationId":     "PROFAPP_001",
						"RequestTrackingId": "RetrieveTBProfileToken-6066477",
					},
					"msg":       "FetchActiveSessionToken process ended",
					"source":    "DataAccess/FetchActiveSessionToken.process",
					"thread":    "JobCourier4",
					"timestamp": "2016-04-05T18:47:18.444Z",
					"error": mapstr.M{
						"message": "type not overwritten (not string)",
						"type":    "json",
					},
				},
			},
		},
		"JSON post processor on invalid struct under type in message": {
			message: reader.Message{
				Content: []byte(`{"timestamp":"2016-04-05T18:47:18.444Z","level":"INFO","logger":"iapi.logger","thread":"JobCourier4","appInfo":{"appname":"SessionManager","appid":"Pooler","host":"demohost.mydomain.com","ip":"192.168.128.113","pid":13982},"userFields":{"ApplicationId":"PROFAPP_001","RequestTrackingId":"RetrieveTBProfileToken-6066477"},"source":"DataAccess\/FetchActiveSessionToken.process","msg":"FetchActiveSessionToken process ended", "type": {"hello": "shouldn't work"}}`),
				Fields:  mapstr.M{},
			},
			config: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"ndjson": map[string]any{
							"target":         "",
							"overwrite_keys": true,
							"add_error_key":  true,
							"message_key":    "msg",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte("FetchActiveSessionToken process ended"),
				Fields: mapstr.M{
					"appInfo": mapstr.M{
						"appname": "SessionManager",
						"appid":   "Pooler",
						"host":    "demohost.mydomain.com",
						"ip":      "192.168.128.113",
						"pid":     int64(13982),
					},
					"level":  "INFO",
					"logger": "iapi.logger",
					"userFields": mapstr.M{
						"ApplicationId":     "PROFAPP_001",
						"RequestTrackingId": "RetrieveTBProfileToken-6066477",
					},
					"msg":       "FetchActiveSessionToken process ended",
					"source":    "DataAccess/FetchActiveSessionToken.process",
					"thread":    "JobCourier4",
					"timestamp": "2016-04-05T18:47:18.444Z",
					"error": mapstr.M{
						"message": "type not overwritten (not string)",
						"type":    "json",
					},
				},
			},
		},
	}

	logger := logptest.NewTestingLogger(t, "")
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.config)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)
			require.NoError(t, err)
			p := c.Create(msgReader(test.message), logger)

			msg, _ := p.Next()
			require.Equal(t, test.expectedMessage, msg)
		})
	}

}

func TestContainerParser(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]any
		expectedMessages []reader.Message
	}{
		"simple docker lines": {
			lines: `{"log":"Fetching main repository github.com/elastic/beats...\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"Fetching dependencies...\n","stream":"stdout","time":"2016-03-02T22:59:04.609292428Z"}
{"log":"Execute /scripts/packetbeat_before_build.sh\n","stream":"stdout","time":"2016-03-02T22:59:04.617434682Z"}
{"log":"patching file vendor/github.com/tsg/gopacket/pcap/pcap.go\n","stream":"stdout","time":"2016-03-02T22:59:04.626534779Z"}
`,
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("Fetching main repository github.com/elastic/beats...\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Fetching dependencies...\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Execute /scripts/packetbeat_before_build.sh\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("patching file vendor/github.com/tsg/gopacket/pcap/pcap.go\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
			},
		},
		"CRI docker lines": {
			lines: `2017-09-12T22:32:21.212861448Z stdout F 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
`,
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{
							"format": "cri",
						},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
			},
		},
		"corrupt docker lines are skipped": {
			lines: `{"log":"Fetching main repository github.com/elastic/beats...\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
"log":"Fetching dependencies...\n","stream":"stdout","time":"2016-03-02T22:59:04.609292428Z"}
{"log":"Execute /scripts/packetbeat_before_build.sh\n","stream":"stdout","time":"2016-03-02T22:59:04.617434682Z"}
`,
			parsers: map[string]any{
				"parsers": []map[string]any{
					map[string]any{
						"container": map[string]any{},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("Fetching main repository github.com/elastic/beats...\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Execute /scripts/packetbeat_before_build.sh\n"),
					Fields: mapstr.M{
						"stream": "stdout",
					},
				},
			},
		},
	}

	logger := logptest.NewTestingLogger(t, "")
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.parsers)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)
			require.NoError(t, err)
			p := c.Create(testReader(test.lines), logger)

			i := 0
			msg, err := p.Next()
			for err == nil {
				require.Equal(t, test.expectedMessages[i].Content, msg.Content)
				require.Equal(t, test.expectedMessages[i].Fields, msg.Fields)
				i++
				msg, err = p.Next()
			}
		})
	}
}

func TestParserIncludeMessages(t *testing.T) {
	parserConfig := map[string]any{
		"parsers": []map[string]any{
			{
				"include_message": map[string]any{
					"patterns": []string{"^INCLUDE"},
				},
			},
		},
	}

	lines := "INCLUDE - FOO\ndo not include this line\n\nINCLUDE BAR\n"
	expectedMessages := []string{
		"INCLUDE - FOO\n",
		"INCLUDE BAR\n",
	}

	cfg := config.MustNewConfigFrom(parserConfig)
	var c inputParsersConfig
	err := cfg.Unpack(&c)
	require.NoError(t, err)

	logger := logptest.NewTestingLogger(t, "")
	p := c.Parsers.Create(testReader(lines), logger)

	readMsgs := []string{}
	msg, err := p.Next()
	for err == nil {
		readMsgs = append(readMsgs, string(msg.Content))
		msg, err = p.Next()
	}

	require.Equal(t, expectedMessages, readMsgs, "fii")
}

type testParsersConfig struct {
	Parsers []config.Namespace `struct:"parsers"`
}

func testReader(lines string) reader.Reader {
	encF, _ := encoding.FindEncoding("")
	reader := strings.NewReader(lines)
	enc, err := encF(reader)
	if err != nil {
		panic(err)
	}
	r, err := readfile.NewEncodeReader(io.NopCloser(reader), readfile.Config{
		Codec:      enc,
		BufferSize: 1024,
		Terminator: readfile.AutoLineTerminator,
		MaxBytes:   1024,
	}, logp.NewNopLogger())
	if err != nil {
		panic(err)
	}

	return r
}

func msgReader(m reader.Message) reader.Reader {
	return &messageReader{
		message: m,
	}
}

type messageReader struct {
	message reader.Message
	read    bool
}

func (r *messageReader) Next() (reader.Message, error) {
	if r.read {
		return reader.Message{}, io.EOF
	}
	r.read = true
	return r.message, nil
}

func (r *messageReader) Close() error {
	r.message = reader.Message{}
	r.read = false
	return nil
}

// reuseCaseSpec is a parser configuration, input, and known-correct output
// exercised by the decode-buffer reuse torture test.
type reuseCaseSpec struct {
	name     string
	parsers  []map[string]interface{} // each map has one key: the parser name
	input    string
	expected []string
}

func dockerLogLine(log string) string {
	return fmt.Sprintf(`{"log":%q,"stream":"stdout","time":"2024-01-01T00:00:00.000000000Z"}`, log) + "\n"
}

// reuseTortureCases lists the parser configs + inputs + expected output
// exercised by the reuse torture test. Platform-specific parsers (auditd) are
// appended by an init() in a build-tagged file. TestReuseTortureCoversAllParsers
// enforces that every supported parser appears here.
var reuseTortureCases = []reuseCaseSpec{
	{"no parsers", nil, "line one\nline two\nline three\n",
		[]string{"line one", "line two", "line three"}},
	{"multiline pattern", []map[string]interface{}{{
		"multiline": map[string]interface{}{"pattern": "^[[:space:]]", "negate": false, "match": "after"},
	}}, "head1\n cont1a\n cont1b\nhead2\n cont2a\nhead3\n",
		[]string{"head1\n cont1a\n cont1b", "head2\n cont2a", "head3"}},
	{"multiline count", []map[string]interface{}{{
		"multiline": map[string]interface{}{"type": "count", "count_lines": 2},
	}}, "a1\na2\nb1\nb2\nc1\nc2\n",
		[]string{"a1\na2", "b1\nb2", "c1\nc2"}},
	{"multiline while", []map[string]interface{}{{
		"multiline": map[string]interface{}{"type": "while_pattern", "pattern": "^[[:space:]]", "negate": true},
	}}, "x1\nx2\nx3\n",
		[]string{"x1\nx2\nx3"}},
	{"ndjson", []map[string]interface{}{{
		"ndjson": map[string]interface{}{"keys_under_root": true, "message_key": "log"},
	}}, dockerLogLine("alpha") + dockerLogLine("beta") + dockerLogLine("gamma"),
		[]string{"alpha", "beta", "gamma"}},
	{"container docker", []map[string]interface{}{{
		"container": map[string]interface{}{"stream": "all", "format": "docker"},
	}}, dockerLogLine("c1\n") + dockerLogLine("c2\n"),
		[]string{"c1\n", "c2\n"}},
	{"container cri partial", []map[string]interface{}{{
		"container": map[string]interface{}{"stream": "all", "format": "cri"},
	}}, "2024-01-01T00:00:00.000000000Z stdout P chunk-one \n" +
		"2024-01-01T00:00:00.000000000Z stdout F chunk-two\n" +
		"2024-01-01T00:00:00.000000000Z stdout F single\n",
		[]string{"chunk-one chunk-two", "single"}},
	{"syslog", []map[string]interface{}{{
		"syslog": map[string]interface{}{"format": "auto"},
	}}, "<13>Oct 11 22:14:15 host app: message one\n<13>Oct 11 22:14:16 host app: message two\n",
		[]string{"message one", "message two"}},
	{"include_message", []map[string]interface{}{{
		"include_message": map[string]interface{}{"patterns": []string{"keep"}},
	}}, "keep one\ndrop two\nkeep three\ndrop four\n",
		[]string{"keep one", "keep three"}},
}

// requiredReuseParsers are the parser names that must each be exercised by
// reuseTortureCases. auditd is appended on linux by the build-tagged init().
var requiredReuseParsers = []string{"multiline", "ndjson", "container", "syslog", "include_message"}

func reuseNamespaces(t *testing.T, specs []map[string]interface{}) []config.Namespace {
	t.Helper()
	if len(specs) == 0 {
		return nil
	}
	cfg := config.MustNewConfigFrom(map[string]interface{}{"parsers": specs})
	var pc struct {
		Parsers []config.Namespace `config:"parsers"`
	}
	require.NoError(t, cfg.Unpack(&pc))
	return pc.Parsers
}

// TestDecodeBufferReuseDoesNotCorrupt is the safety net for decode buffer
// reuse, which streambuf.Buffer now always performs. For each parser, the
// produced messages must match a known-correct golden output even though the
// line reader reuses the array backing each line's Content across reads. A
// parser that holds a reference into that buffer across reads would corrupt
// its output and fail this test. A tiny BufferSize maximizes buffer churn to
// stress it.
func TestDecodeBufferReuseDoesNotCorrupt(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	readAll := func(t *testing.T, parsers []config.Namespace, input string) []string {
		t.Helper()
		c, err := NewConfig(CommonConfig{MaxBytes: 1 << 20, LineTerminator: readfile.AutoLineTerminator}, parsers)
		require.NoError(t, err)

		encF, _ := encoding.FindEncoding("")
		sr := strings.NewReader(input)
		enc, err := encF(sr)
		require.NoError(t, err)
		// Wrap the source so it honors read deadlines, mirroring filestream's file
		// reader. This makes the multiline timeout reader use its synchronous,
		// goroutine-free path (the path used in production), which is the one that
		// must be safe under decode-buffer reuse.
		// Tiny BufferSize forces frequent reads and thus maximal decode-buffer reuse.
		er, err := readfile.NewEncodeReader(deadlineNopCloser{io.NopCloser(sr)}, readfile.Config{
			Codec:      enc,
			BufferSize: 4,
			Terminator: readfile.LineFeed,
			MaxBytes:   1 << 20,
		}, logger)
		require.NoError(t, err)
		r := c.Create(readfile.NewStripNewline(er, readfile.LineFeed), logger)

		var out []string
		for {
			msg, err := r.Next()
			if msg.Bytes > 0 || len(msg.Content) > 0 {
				out = append(out, string(msg.Content)) // copy + retain, like the harvester
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					require.NoError(t, err)
				}
				break
			}
		}
		return out
	}

	for _, tc := range reuseTortureCases {
		t.Run(tc.name, func(t *testing.T) {
			parsers := reuseNamespaces(t, tc.parsers)
			out := readAll(t, parsers, tc.input)
			require.Equal(t, tc.expected, out, "decode-buffer reuse corrupted parser output")
		})
	}
}

// TestReuseTortureCoversAllParsers enforces that every supported parser is
// exercised by the decode-buffer reuse torture test, so a newly added parser
// cannot silently skip the safety net.
func TestReuseTortureCoversAllParsers(t *testing.T) {
	covered := map[string]bool{}
	for _, c := range reuseTortureCases {
		for _, m := range c.parsers {
			for name := range m {
				covered[name] = true
			}
		}
	}
	for _, name := range requiredReuseParsers {
		require.Truef(t, covered[name],
			"parser %q is not exercised by TestDecodeBufferReuseDoesNotCorrupt; add a case to reuseTortureCases", name)
	}

	// Guard the requiredReuseParsers list against drift: an unknown parser must
	// be rejected, confirming NewConfig's accepted set is enumerable here.
	_, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator},
		reuseNamespaces(t, []map[string]interface{}{{"definitely_not_a_parser": map[string]interface{}{}}}))
	require.ErrorIs(t, err, ErrNoSuchParser)
}

// deadlineNopCloser makes an io.ReadCloser honor read deadlines (trivially: an
// in-memory source never blocks, so the deadline never fires), so tests can
// exercise the synchronous, goroutine-free timeout path used in production.
type deadlineNopCloser struct{ io.ReadCloser }

func (deadlineNopCloser) SetReadDeadline(time.Time) bool { return true }

// deadlineMockReader is a reader.Reader that records whether SetReadDeadline was
// forwarded to it.
type deadlineMockReader struct{ gotDeadline bool }

func (m *deadlineMockReader) Next() (reader.Message, error) { return reader.Message{}, io.EOF }
func (m *deadlineMockReader) Close() error                  { return nil }
func (m *deadlineMockReader) SetReadDeadline(time.Time) bool {
	m.gotDeadline = true
	return true
}

// TestParsersForwardReadDeadline ensures every parser that can sit below
// multiline forwards SetReadDeadline to the reader it wraps. The multiline
// timeout is enforced synchronously via read deadlines (no goroutine); a parser
// that fails to forward the deadline would leave multiline unable to time out,
// so it would block forever waiting for the source. This guards against that by
// asserting the deadline reaches the wrapped source through each parser.
func TestParsersForwardReadDeadline(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	for _, tc := range reuseTortureCases {
		// "no parsers" has nothing to wrap; multiline hosts the timeout itself and
		// is never positioned below another reader's deadline.
		if len(tc.parsers) == 0 || strings.HasPrefix(tc.name, "multiline") {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			parsers := reuseNamespaces(t, tc.parsers)
			c, err := NewConfig(CommonConfig{MaxBytes: 1 << 20, LineTerminator: readfile.AutoLineTerminator}, parsers)
			require.NoError(t, err)

			mock := &deadlineMockReader{}
			p := c.Create(mock, logger)
			ok := reader.SetReadDeadline(p, time.Now().Add(time.Second))
			require.Truef(t, ok, "parser chain %q did not forward SetReadDeadline to its source", tc.name)
			require.Truef(t, mock.gotDeadline, "parser chain %q did not call SetReadDeadline on its source", tc.name)
		})
	}
}
