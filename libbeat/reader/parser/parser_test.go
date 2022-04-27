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
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/multiline"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestParsersConfigSuffix(t *testing.T) {
	tests := map[string]struct {
		parsers        map[string]interface{}
		expectedSuffix string
		expectedError  string
	}{
		"parsers with no suffix config": {
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{
							"stream": "all",
						},
					},
				},
			},
		},
		"parsers with correct suffix config": {
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{
							"stream": "stdout",
						},
					},
				},
			},
			expectedSuffix: "stdout",
		},
		"parsers with multiple suffix config": {
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{
							"stream": "stdout",
						},
					},
					map[string]interface{}{
						"container": map[string]interface{}{
							"stream": "stderr",
						},
					},
				},
			},
			expectedError: "only one stream selection is allowed",
		},
	}

	for name, test := range tests {
		test := test
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
			require.Equal(t, c.Suffix, test.expectedSuffix)
		})
	}

}

func TestParsersConfigAndReading(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]interface{}
		expectedMessages []string
		expectedError    string
	}{
		"no parser, no error": {
			lines:            "line 1\nline 2\n",
			parsers:          map[string]interface{}{},
			expectedMessages: []string{"line 1\n", "line 2\n"},
		},
		"correct multiline parser": {
			lines: "line 1.1\nline 1.2\nline 1.3\nline 2.1\nline 2.2\nline 2.3\n",
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"multiline": map[string]interface{}{
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
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"keys_under_root": true,
							"message_key":     "log",
						},
					},
					map[string]interface{}{
						"multiline": map[string]interface{}{
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
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"no_such_parser": nil,
					},
				},
			},
			expectedError: ErrNoSuchParser.Error(),
		},
		"invalid multiline parser configuration is caught before parser creation": {
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"multiline": map[string]interface{}{
							"match": "after",
						},
					},
				},
			},
			expectedError: multiline.ErrMissingPattern.Error(),
		},
		"ndjson with syslog": {
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					{
						"ndjson": map[string]interface{}{
							"keys_under_root": true,
							"message_key":     "log",
						},
					},
					{
						"syslog": map[string]interface{}{
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
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					{
						"multiline": map[string]interface{}{
							"match":        "after",
							"negate":       true,
							"pattern":      "^<\\d{1,3}>",
							"skip_newline": true, // This option is set since testReader does not strip newlines when splitting lines.
						},
					},
					{
						"syslog": map[string]interface{}{
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
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					{
						"syslog": map[string]interface{}{
							"format": "rfc5424",
						},
					},
					{
						"multiline": map[string]interface{}{
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

	for name, test := range tests {
		test := test
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

			p := c.Create(testReader(test.lines))

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
		config          map[string]interface{}
		expectedMessage reader.Message
	}{
		"no postprocesser, no processing": {
			message: reader.Message{
				Content: []byte("line 1"),
			},
			config: map[string]interface{}{},
			expectedMessage: reader.Message{
				Content: []byte("line 1"),
			},
		},
		"JSON post processer with keys_under_root": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\"}"),
				Fields:  common.MapStr{},
			},
			config: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target": "",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key": "value",
				},
			},
		},
		"JSON post processer with document ID": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\", \"my-id-field\":\"my-id\"}"),
				Fields:  common.MapStr{},
			},
			config: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target":      "",
							"document_id": "my-id-field",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key": "value",
				},
				Meta: common.MapStr{
					"_id": "my-id",
				},
			},
		},
		"JSON post processer with overwrite keys and under root": {
			message: reader.Message{
				Content: []byte("{\"key\": \"value\"}"),
				Fields: common.MapStr{
					"key":       "another-value",
					"other-key": "other-value",
				},
			},
			config: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target":         "",
							"overwrite_keys": true,
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key":       "value",
					"other-key": "other-value",
				},
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.config)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)
			require.NoError(t, err)
			p := c.Create(msgReader(test.message))

			msg, _ := p.Next()
			require.Equal(t, test.expectedMessage, msg)
		})
	}

}

func TestContainerParser(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]interface{}
		expectedMessages []reader.Message
	}{
		"simple docker lines": {
			lines: `{"log":"Fetching main repository github.com/elastic/beats...\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"Fetching dependencies...\n","stream":"stdout","time":"2016-03-02T22:59:04.609292428Z"}
{"log":"Execute /scripts/packetbeat_before_build.sh\n","stream":"stdout","time":"2016-03-02T22:59:04.617434682Z"}
{"log":"patching file vendor/github.com/tsg/gopacket/pcap/pcap.go\n","stream":"stdout","time":"2016-03-02T22:59:04.626534779Z"}
`,
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("Fetching main repository github.com/elastic/beats...\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Fetching dependencies...\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Execute /scripts/packetbeat_before_build.sh\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("patching file vendor/github.com/tsg/gopacket/pcap/pcap.go\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
			},
		},
		"CRI docker lines": {
			lines: `2017-09-12T22:32:21.212861448Z stdout F 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
`,
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{
							"format": "cri",
						},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache\n"),
					Fields: common.MapStr{
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
			parsers: map[string]interface{}{
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"container": map[string]interface{}{},
					},
				},
			},
			expectedMessages: []reader.Message{
				reader.Message{
					Content: []byte("Fetching main repository github.com/elastic/beats...\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
				reader.Message{
					Content: []byte("Execute /scripts/packetbeat_before_build.sh\n"),
					Fields: common.MapStr{
						"stream": "stdout",
					},
				},
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.parsers)
			var parsersConfig testParsersConfig
			err := cfg.Unpack(&parsersConfig)
			require.NoError(t, err)
			c, err := NewConfig(CommonConfig{MaxBytes: 1024, LineTerminator: readfile.AutoLineTerminator}, parsersConfig.Parsers)
			require.NoError(t, err)
			p := c.Create(testReader(test.lines))

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
	r, err := readfile.NewEncodeReader(ioutil.NopCloser(reader), readfile.Config{
		Codec:      enc,
		BufferSize: 1024,
		Terminator: readfile.AutoLineTerminator,
		MaxBytes:   1024,
	})
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
