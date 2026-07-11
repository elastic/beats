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

package tls

import (
	"encoding/hex"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/elastic-agent-libs/logp"
)

func newBenchParser() *parser {
	return &parser{logger: logp.NewNopLogger()}
}

// BenchmarkCertificatePipeline simulates a persistent TLS connection where a
// full certificate handshake record (two certificates, ~2.7KB) arrives
// complete in a single read, repeated many times on one parser/handshakeBuf.
func BenchmarkCertificatePipeline(b *testing.B) {
	raw, err := hex.DecodeString(certsMsg)
	if err != nil {
		b.Fatalf("failed to decode certsMsg: %v", err)
	}

	p := newBenchParser()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := streambuf.New(raw)
		if res := p.parse(buf); res != resultOK {
			b.Fatalf("unexpected parse result: %v", res)
		}
	}
}

// wrapRecord wraps a handshake-message chunk in its own TLS 1.2 handshake
// record header (type 0x16, version 0x0303, 2-byte length).
func wrapRecord(chunk []byte) []byte {
	rec := make([]byte, 0, 5+len(chunk))
	rec = append(rec, 0x16, 0x03, 0x03, byte(len(chunk)>>8), byte(len(chunk)))
	return append(rec, chunk...)
}

// BenchmarkCertificatePipelineFragmented simulates the same persistent
// connection, but with the certificate handshake message split across two
// TLS records (a large handshake message like Certificate routinely spans
// several records), which is the scenario parser.handshakeBuf exists to
// reassemble: two Append+parse cycles into handshakeBuf before it completes
// and Resets, instead of one.
func BenchmarkCertificatePipelineFragmented(b *testing.B) {
	raw, err := hex.DecodeString(certsMsg)
	if err != nil {
		b.Fatalf("failed to decode certsMsg: %v", err)
	}
	// raw is a single record: 5-byte record header + handshake message body.
	handshakeMsg := raw[5:]
	split := len(handshakeMsg) / 2
	record1 := wrapRecord(handshakeMsg[:split])
	record2 := wrapRecord(handshakeMsg[split:])

	p := newBenchParser()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.certificates = nil

		if res := p.parse(streambuf.New(record1)); res != resultOK {
			b.Fatalf("unexpected parse result for record1: %v", res)
		}
		if p.certificates != nil {
			b.Fatalf("certificates parsed before handshake message completed")
		}
		if res := p.parse(streambuf.New(record2)); res != resultOK {
			b.Fatalf("unexpected parse result for record2: %v", res)
		}
		if len(p.certificates) != 2 {
			b.Fatalf("expected 2 certificates, got %d", len(p.certificates))
		}
	}
}
