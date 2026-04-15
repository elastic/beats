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
	"bytes"
	stdjson "encoding/json"
	"io"
	"os"
	"testing"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

// benchNDJSONLines are the two log lines that make up the benchmark file,
// alternated across 1000 lines. Kept here so the file benchmark reflects the
// same inputs as BenchmarkJSONPipelineE2E in the readjson package.
var (
	benchMediumLine   = []byte(`{"message":"GET /api/users 200","level":"info","timestamp":"2024-01-15T10:30:00Z","duration":142,"method":"GET","path":"/api/users","status":200,"bytes_sent":1024,"user_agent":"Mozilla/5.0","remote_addr":"10.0.0.1"}`)
	benchJournaldLine = []byte(`{"message":"pam_unix(sudo:session): session closed for user root","event":{"kind":"event"},"host":{"hostname":"x-wing","id":"a6a19d57efcf4bf38705c63217a63ba3"},"journald":{"audit":{"login_uid":1000,"session":"1"},"custom":{"syslog_timestamp":"Nov 22 18:10:04 "},"gid":0,"host":{"boot_id":"537d392f028b4dd4b9b1995a4c78cfb6"},"pid":2084586,"process":{"capabilities":"1ffffffffff","command_line":"sudo journalctl --user --rotate","executable":"/usr/bin/sudo","name":"sudo"},"uid":1000},"log":{"syslog":{"appname":"sudo","facility":{"code":10},"priority":6}},"process":{"args":["sudo","journalctl","--user","--rotate"],"args_count":4,"command_line":"sudo journalctl --user --rotate","pid":2084586,"thread":{"capabilities":{"effective":["CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_DAC_READ_SEARCH","CAP_FOWNER","CAP_FSETID","CAP_KILL","CAP_SETGID","CAP_SETUID"]}}}}`)
)

// BenchmarkNDJSONFilePipeline benchmarks the real filestream ndjson pipeline:
// os.File → readfile.NewEncodeReader → readfile.NewStripNewline → parser.Config.Create(ndjson)
//
// This is the production code path. Each iteration opens a 1000-line NDJSON
// file and drains it through the pipeline. Throughput is in MB/s.
//
// The stdlib baseline drives the same file reader chain but calls
// json.NewDecoder per line (the old decode() behaviour before this PR).
func BenchmarkNDJSONFilePipeline(b *testing.B) {
	// Build the parser config once, as filestream does at startup.
	parserCfg := &Config{}
	if err := conf.MustNewConfigFrom(map[string]interface{}{
		"parsers": []map[string]interface{}{
			{"ndjson": map[string]interface{}{}},
		},
	}).Unpack(parserCfg); err != nil {
		b.Fatal(err)
	}

	// Write a 1000-line NDJSON temp file.
	const nLines = 1000
	f, err := os.CreateTemp(b.TempDir(), "bench-*.ndjson")
	if err != nil {
		b.Fatal(err)
	}
	fname := f.Name()
	var total int
	for i := 0; i < nLines; i++ {
		line := benchMediumLine
		if i%2 != 0 {
			line = benchJournaldLine
		}
		n, _ := f.Write(line)
		f.Write([]byte{'\n'})
		total += n + 1
	}
	f.Close()

	encF, _ := encoding.FindEncoding("")
	logger := logp.NewNopLogger()
	readerCfg := readfile.Config{
		BufferSize: 16 * 1024,
		Terminator: readfile.AutoLineTerminator,
		MaxBytes:   10 * 1024 * 1024,
	}

	// openPipeline opens the file and returns the full reader chain.
	// The caller is responsible for closing the returned file.
	openPipeline := func(b *testing.B) (*os.File, Parser) {
		b.Helper()
		f, err := os.Open(fname)
		if err != nil {
			b.Fatal(err)
		}
		codec, err := encF(f)
		if err != nil {
			f.Close()
			b.Fatal(err)
		}
		cfg := readerCfg
		cfg.Codec = codec
		enc, err := readfile.NewEncodeReader(f, cfg, logger)
		if err != nil {
			f.Close()
			b.Fatal(err)
		}
		strip := readfile.NewStripNewline(enc, readfile.AutoLineTerminator)
		return f, parserCfg.Create(strip, logger)
	}

	b.Run("0_stdlib_baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(total))
		for b.Loop() {
			f, err := os.Open(fname)
			if err != nil {
				b.Fatal(err)
			}
			codec, _ := encF(f)
			cfg := readerCfg
			cfg.Codec = codec
			enc, _ := readfile.NewEncodeReader(f, cfg, logger)
			strip := readfile.NewStripNewline(enc, readfile.AutoLineTerminator)
			for {
				msg, err := strip.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					b.Fatal(err)
				}
				dec := stdjson.NewDecoder(bytes.NewReader(msg.Content))
				dec.UseNumber()
				var fields map[string]interface{}
				if err := dec.Decode(&fields); err == nil {
					jsontransform.TransformNumbers(fields)
				}
			}
			f.Close()
		}
	})

	b.Run("1_jsoniter_pipeline", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(total))
		for b.Loop() {
			f, pipeline := openPipeline(b)
			for {
				_, err := pipeline.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					b.Fatal(err)
				}
			}
			f.Close()
		}
	})
}
