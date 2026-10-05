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

package iobuf

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
)

func ExampleReadAll() {
	r := strings.NewReader("The quick brown fox jumps over the lazy dog.")

	b, err := ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", b)

	// Output:
	// The quick brown fox jumps over the lazy dog.
}

// dumbReadSeeker is a ReadSeeker that does not implement the io.WriteTo optimization.
type dumbReadSeeker struct {
	rs io.ReadSeeker
}

func (r *dumbReadSeeker) Read(p []byte) (n int, err error) {
	return r.rs.Read(p)
}

func (r *dumbReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return r.rs.Seek(offset, whence)
}

func genData(n int) io.ReadSeeker {
	return bytes.NewReader(bytes.Repeat([]byte{'a'}, n))
}

func genDataDumb(n int) io.ReadSeeker {
	return &dumbReadSeeker{rs: genData(n)}
}

func BenchmarkReadAll(b *testing.B) {
	// Make sure we test different sizes to overcome initial buffer sizes:
	// 	io.ReadAll uses a 512 bytes buffer
	// 	bytes.Buffer uses a 64 bytes buffer
	sizes := []int{
		32,          // 32 bytes
		64,          // 64 bytes
		512,         // 512 bytes
		10 * 1024,   // 10KB
		100 * 1024,  // 100KB
		1024 * 1024, // 1MB
	}
	sizesReadable := []string{
		"32B",
		"64B",
		"512B",
		"10KB",
		"100KB",
		"1MB",
	}

	benchFunc := func(b *testing.B, size int, genFunc func(n int) io.ReadSeeker, readFunc func(io.Reader) ([]byte, error)) {
		buf := genFunc(size)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := buf.Seek(0, io.SeekStart) // reset
			if err != nil {
				b.Fatal(err)
			}
			data, err := readFunc(buf)
			if err != nil {
				b.Fatal(err)
			}
			if len(data) != size {
				b.Fatalf("size does not match, expected %d, actual %d", size, len(data))
			}
		}
	}

	for i, size := range sizes {
		b.Run(fmt.Sprintf("size %s", sizesReadable[i]), func(b *testing.B) {
			b.Run("io.ReadAll", func(b *testing.B) {
				b.Run("WriterTo", func(b *testing.B) {
					benchFunc(b, size, genData, io.ReadAll)
				})
				b.Run("Reader", func(b *testing.B) {
					benchFunc(b, size, genDataDumb, io.ReadAll)
				})
			})
			b.Run("ReadAll", func(b *testing.B) {
				b.Run("WriterTo", func(b *testing.B) {
					benchFunc(b, size, genData, ReadAll)
				})
				b.Run("Reader", func(b *testing.B) {
					benchFunc(b, size, genDataDumb, ReadAll)
				})
			})
		})
	}
}
