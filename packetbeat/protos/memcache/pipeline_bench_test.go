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

//go:build !integration

package memcache

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

func newBenchStream() *stream {
	logger := logp.NewNopLogger()
	st := &stream{
		logger: logger,
		parser: *newParser(&defaultTestParserConfig, logger),
	}
	st.Stream.Init(0)
	return st
}

// drain feeds one complete message's worth of bytes to the stream and parses
// until the message yields, mirroring memcacheParseTCP's loop.
func drain(b *testing.B, st *stream) {
	for st.Buf.Total() > 0 {
		msg, err := st.parse(time.Time{})
		if err != nil {
			b.Fatalf("parse error: %v", err)
		}
		if msg == nil {
			return
		}
		st.reset()
	}
}

// BenchmarkTextPipeline simulates a persistent connection processing many
// complete text-protocol commands, one full message per Append (the common
// case: a command fits in a single TCP read), cycling through Reset+Append on
// one stream.
func BenchmarkTextPipeline(b *testing.B) {
	cmds := [][]byte{
		[]byte("get key\r\n"),
		[]byte("set key1 0 0 5\r\nHello\r\n"),
		[]byte("VALUE key1 0 5\r\nHello\r\nEND\r\n"),
	}

	st := newBenchStream()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = st.Append(cmds[i%len(cmds)])
		drain(b, st)
	}
}

// BenchmarkTextPipelineFragmented simulates a persistent connection where
// every text command arrives split across two TCP segments, so the buffer is
// non-empty across an Append boundary.
func BenchmarkTextPipelineFragmented(b *testing.B) {
	part1 := []byte("set key1 0 0 5")
	part2 := []byte("\r\nHello\r\n")

	st := newBenchStream()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = st.Append(part1)
		drain(b, st)

		_ = st.Append(part2)
		drain(b, st)
	}
}

// BenchmarkBinaryPipeline is the binary-protocol equivalent of
// BenchmarkTextPipeline: one complete request per Append.
func BenchmarkBinaryPipeline(b *testing.B) {
	setBuf, err := prepareBinMessage(
		&binHeader{opcode: opcodeSet, request: true},
		extras(extra32Bit(0x1f2f), extra32Bit(0x11223344)),
		key("key1"),
		value("Hello"))
	if err != nil {
		b.Fatalf("failed to prepare set message: %v", err)
	}
	getBuf, err := prepareBinMessage(
		&binHeader{opcode: opcodeGet, request: true},
		extras(), key("key1"), noValue)
	if err != nil {
		b.Fatalf("failed to prepare get message: %v", err)
	}
	cmds := [][]byte{setBuf.Bytes(), getBuf.Bytes()}

	st := newBenchStream()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = st.Append(cmds[i%len(cmds)])
		drain(b, st)
	}
}

// BenchmarkBinaryPipelineFragmented is the binary-protocol equivalent of
// BenchmarkTextPipelineFragmented: one request split across two Append calls.
func BenchmarkBinaryPipelineFragmented(b *testing.B) {
	setBuf, err := prepareBinMessage(
		&binHeader{opcode: opcodeSet, request: true},
		extras(extra32Bit(0x1f2f), extra32Bit(0x11223344)),
		key("key1"),
		value("Hello"))
	if err != nil {
		b.Fatalf("failed to prepare set message: %v", err)
	}
	full := setBuf.Bytes()
	split := len(full) / 2
	part1, part2 := full[:split], full[split:]

	st := newBenchStream()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = st.Append(part1)
		drain(b, st)

		_ = st.Append(part2)
		drain(b, st)
	}
}
