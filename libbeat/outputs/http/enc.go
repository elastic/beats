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

package http

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
)

type bodyEncoder interface {
	bulkBodyEncoder
	Reader() io.Reader
	Marshal(doc interface{}) error
}

type bulkBodyEncoder interface {
	bulkWriter

	AddHeader(*http.Header, string)
	Reset()
}

type bulkWriter interface {
	Add(meta, obj interface{}) error
	AddRaw(raw interface{}) error
}

type jsonEncoder struct {
	buf *bytes.Buffer
}

type jsonLinesEncoder struct {
	buf *bytes.Buffer
}

type gzipEncoder struct {
	buf  *bytes.Buffer
	gzip *gzip.Writer
}

type gzipLinesEncoder struct {
	buf  *bytes.Buffer
	gzip *gzip.Writer
}

func newJSONEncoder(buf *bytes.Buffer) *jsonEncoder {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	return &jsonEncoder{buf}
}

func (b *jsonEncoder) Reset() {
	b.buf.Reset()
}

func (b *jsonEncoder) AddHeader(header *http.Header, contentType string) {
	if contentType == "" {
		header.Add("Content-Type", "application/json; charset=UTF-8")
	} else {
		header.Add("Content-Type", contentType)
	}
}

func (b *jsonEncoder) Reader() io.Reader {
	return b.buf
}

func (b *jsonEncoder) Marshal(obj interface{}) error {
	b.Reset()
	enc := json.NewEncoder(b.buf)
	return enc.Encode(obj)
}

func (b *jsonEncoder) AddRaw(raw interface{}) error {
	enc := json.NewEncoder(b.buf)
	return enc.Encode(raw)
}

func (b *jsonEncoder) Add(meta, obj interface{}) error {
	enc := json.NewEncoder(b.buf)
	pos := b.buf.Len()

	if err := enc.Encode(meta); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	if err := enc.Encode(obj); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	return nil
}

func newJSONLinesEncoder(buf *bytes.Buffer) *jsonLinesEncoder {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	return &jsonLinesEncoder{buf}
}

func (b *jsonLinesEncoder) Reset() {
	b.buf.Reset()
}

func (b *jsonLinesEncoder) AddHeader(header *http.Header, contentType string) {
	if contentType == "" {
		header.Add("Content-Type", "application/x-ndjson; charset=UTF-8")
	} else {
		header.Add("Content-Type", contentType)
	}
}

func (b *jsonLinesEncoder) Reader() io.Reader {
	return b.buf
}

func (b *jsonLinesEncoder) Marshal(obj interface{}) error {
	b.Reset()
	return b.AddRaw(obj)
}

func (b *jsonLinesEncoder) AddRaw(obj interface{}) error {
	enc := json.NewEncoder(b.buf)

	// single event
	if reflect.TypeOf(obj).Kind() == reflect.Map {
		return enc.Encode(obj)
	}

	// batch of events
	for _, item := range obj.([]eventRaw) {
		err := enc.Encode(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *jsonLinesEncoder) Add(meta, obj interface{}) error {
	pos := b.buf.Len()

	if err := b.AddRaw(meta); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	if err := b.AddRaw(obj); err != nil {
		b.buf.Truncate(pos)
		return err
	}

	return nil
}

func newGzipEncoder(level int, buf *bytes.Buffer) (*gzipEncoder, error) {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	w, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, err
	}

	return &gzipEncoder{buf, w}, nil
}

func (b *gzipEncoder) Reset() {
	b.buf.Reset()
	b.gzip.Reset(b.buf)
}

func (b *gzipEncoder) Reader() io.Reader {
	b.gzip.Close()
	return b.buf
}

func (b *gzipEncoder) AddHeader(header *http.Header, contentType string) {
	if contentType == "" {
		header.Add("Content-Type", "application/json; charset=UTF-8")
	} else {
		header.Add("Content-Type", contentType)
	}
	header.Add("Content-Encoding", "gzip")
}

func (b *gzipEncoder) Marshal(obj interface{}) error {
	b.Reset()
	enc := json.NewEncoder(b.gzip)
	err := enc.Encode(obj)
	return err
}

func (b *gzipEncoder) AddRaw(raw interface{}) error {
	enc := json.NewEncoder(b.gzip)
	return enc.Encode(raw)
}

func (b *gzipEncoder) Add(meta, obj interface{}) error {
	enc := json.NewEncoder(b.gzip)
	pos := b.buf.Len()

	if err := enc.Encode(meta); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	if err := enc.Encode(obj); err != nil {
		b.buf.Truncate(pos)
		return err
	}

	b.gzip.Flush()
	return nil
}

func newGzipLinesEncoder(level int, buf *bytes.Buffer) (*gzipLinesEncoder, error) {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	w, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, err
	}

	return &gzipLinesEncoder{buf, w}, nil
}

func (b *gzipLinesEncoder) Reset() {
	b.buf.Reset()
	b.gzip.Reset(b.buf)
}

func (b *gzipLinesEncoder) Reader() io.Reader {
	b.gzip.Close()
	return b.buf
}

func (b *gzipLinesEncoder) AddHeader(header *http.Header, contentType string) {
	if contentType == "" {
		header.Add("Content-Type", "application/x-ndjson; charset=UTF-8")
	} else {
		header.Add("Content-Type", contentType)
	}
	header.Add("Content-Encoding", "gzip")
}

func (b *gzipLinesEncoder) Marshal(obj interface{}) error {
	b.Reset()
	return b.AddRaw(obj)
}

func (b *gzipLinesEncoder) AddRaw(obj interface{}) error {
	enc := json.NewEncoder(b.gzip)

	// single event
	if reflect.TypeOf(obj).Kind() == reflect.Map {
		return enc.Encode(obj)
	}

	// batch of events
	for _, item := range obj.([]eventRaw) {
		err := enc.Encode(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *gzipLinesEncoder) Add(meta, obj interface{}) error {
	pos := b.buf.Len()

	if err := b.AddRaw(meta); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	if err := b.AddRaw(obj); err != nil {
		b.buf.Truncate(pos)
		return err
	}

	b.gzip.Flush()
	return nil
}
