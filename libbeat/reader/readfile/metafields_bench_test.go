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

package readfile

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// freshMsgReader returns the same message on every Next() call with a freshly
// allocated Fields map, so benchmarks measure only FileMetaReader.Next() overhead.
type freshMsgReader struct {
	msg reader.Message
}

func (r *freshMsgReader) Next() (reader.Message, error) {
	r.msg.Fields = make(mapstr.M, 1)
	return r.msg, nil
}

func (r *freshMsgReader) Close() error { return nil }

func BenchmarkFileMetaReaderNext(b *testing.B) {
	fi := createTestFileInfo()
	base := reader.Message{
		Content: []byte("2024-01-01T00:00:00Z INFO example log line with some content"),
		Bytes:   60,
	}

	b.Run("no-fingerprint", func(b *testing.B) {
		r := &FileMetaReader{
			reader: &freshMsgReader{msg: base},
			path:   "/var/log/app/test.log",
			fi:     fi,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := r.Next(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("with-fingerprint", func(b *testing.B) {
		r := &FileMetaReader{
			reader:      &freshMsgReader{msg: base},
			path:        "/var/log/app/test.log",
			fi:          fi,
			fingerprint: "abc123deadbeef0123456789abcdef01",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := r.Next(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
