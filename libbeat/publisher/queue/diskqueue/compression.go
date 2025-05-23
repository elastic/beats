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

package diskqueue

import (
	"io"

	lz4V4 "github.com/pierrec/lz4/v4"
)

// CompressionReader allows reading a stream compressed with LZ4
type CompressionReader struct {
	src        io.ReadCloser
	pLZ4Reader *lz4V4.Reader
}

// NewCompressionReader returns a new LZ4 frame decoder
func NewCompressionReader(r io.ReadCloser) *CompressionReader {
	zr := lz4V4.NewReader(r)
	return &CompressionReader{
		src:        r,
		pLZ4Reader: zr,
	}
}

func (r *CompressionReader) Read(buf []byte) (int, error) {
	return r.pLZ4Reader.Read(buf)
}

func (r *CompressionReader) Close() error {
	return r.src.Close()
}

// Reset Sets up compression again, assumes that caller has already set
// the src to the correct position
func (r *CompressionReader) Reset() error {
	r.pLZ4Reader.Reset(r.src)
	return nil
}

// CompressionWriter allows writing an LZ4 stream
type CompressionWriter struct {
	dst        WriteCloseSyncer
	pLZ4Writer *lz4V4.Writer
}

// NewCompressionWriter returns a new LZ4 frame encoder
func NewCompressionWriter(w WriteCloseSyncer) *CompressionWriter {
	zw := lz4V4.NewWriter(w)
	return &CompressionWriter{
		dst:        w,
		pLZ4Writer: zw,
	}
}

func (w *CompressionWriter) Write(p []byte) (int, error) {
	return w.pLZ4Writer.Write(p)
}

func (w *CompressionWriter) Close() error {
	err := w.pLZ4Writer.Close()
	if err != nil {
		return err
	}
	return w.dst.Close()
}

func (w *CompressionWriter) Sync() error {
	w.pLZ4Writer.Flush()
	return w.dst.Sync()
}
