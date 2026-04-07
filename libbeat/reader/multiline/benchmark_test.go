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

package multiline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
)

// BenchmarkPatternReaderRabbitMQLike benchmarks the multiline reader.
// This benchmark was added to assess the cost of adding a mutex to protect
// the multiline reader's state field. The results:
//
// goos: linux
// goarch: amd64
// pkg: github.com/elastic/beats/v7/libbeat/reader/multiline
// cpu: Intel(R) Core(TM) Ultra 9 285H
//
//	│ before-10.txt │            after-10.txt            │
//	│    sec/op     │   sec/op     vs base               │
//
// PatternReaderRabbitMQLike-16     750.5µ ± 7%   714.8µ ± 5%  -4.75% (p=0.035 n=10)
//
//	│ before-10.txt │            after-10.txt             │
//	│      B/s      │     B/s       vs base               │
//
// PatternReaderRabbitMQLike-16    63.11Mi ± 7%   66.26Mi ± 5%  +4.99% (p=0.035 n=10)
//
//	│ before-10.txt │          after-10.txt          │
//	│     B/op      │     B/op      vs base          │
//
// PatternReaderRabbitMQLike-16    557.6Ki ± 0%   557.8Ki ± 0%  ~ (p=0.579 n=10)
//
//	│ before-10.txt │          after-10.txt           │
//	│   allocs/op   │  allocs/op   vs base            │
//
// PatternReaderRabbitMQLike-16     5.680k ± 0%   5.680k ± 0%  ~ (p=1.000 n=10) ¹
// ¹ all samples are equal
func BenchmarkPatternReaderRabbitMQLike(b *testing.B) {
	const (
		eventCount = 256
		maxBytes   = 1 << 20 // 1MiB
	)

	input := benchmarkRabbitMQInput(eventCount)
	pattern := match.MustCompile("^=[A-Z]+")
	timeout := time.Duration(0)
	cfg := Config{
		Type:    patternMode,
		Pattern: &pattern,
		Negate:  true,
		Match:   "after",
		Timeout: &timeout,
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for b.Loop() {
		r := newBenchmarkMultilineReader(b, input, cfg, maxBytes)
		events := drainBenchmarkReader(b, r)
		if events != eventCount {
			b.Fatalf("expected %d events but got %d", eventCount, events)
		}

		if err := r.Close(); err != nil {
			b.Fatalf("unexpected close error: %v", err)
		}
	}
}

func newBenchmarkMultilineReader(b *testing.B, input []byte, cfg Config, maxBytes int) reader.Reader {
	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		b.Fatal("unable to find plain encoding")
	}

	enc, err := encFactory(bytes.NewReader(input))
	if err != nil {
		b.Fatalf("failed to initialize encoding: %v", err)
	}

	encReader, err := readfile.NewEncodeReader(
		io.NopCloser(bytes.NewReader(input)),
		readfile.Config{
			Codec:      enc,
			BufferSize: 4096,
			Terminator: readfile.LineFeed,
		},
		logp.NewNopLogger(),
	)
	if err != nil {
		b.Fatalf("failed to initialize encode reader: %v", err)
	}

	mlReader, err := New(
		readfile.NewStripNewline(encReader, readfile.LineFeed),
		"\n",
		maxBytes,
		&cfg,
		logp.NewNopLogger(),
	)
	if err != nil {
		b.Fatalf("failed to initialize multiline reader: %v", err)
	}

	return mlReader
}

func drainBenchmarkReader(b *testing.B, r reader.Reader) int {
	events := 0
	for {
		_, err := r.Next()
		if err == nil {
			events++
			continue
		}
		if errors.Is(err, io.EOF) {
			return events
		}

		b.Fatalf("unexpected read error after %d events: %v", events, err)
	}
}

func benchmarkRabbitMQInput(eventCount int) []byte {
	var input bytes.Buffer

	for i := range eventCount {
		fmt.Fprintf(&input, "=ERROR REPORT==== 3-Feb-2016::03:10:%02d ===\n", i%60)
		input.WriteString("connection <0.23893.109>, channel 3 - soft error:\n")
		input.WriteString("{amqp_error,not_found,\n")
		input.WriteString("            \"no queue 'bucket-1' in vhost '/'\",\n")
		input.WriteString("            'queue.declare'}\n")
		input.WriteByte('\n')
	}

	return input.Bytes()
}
