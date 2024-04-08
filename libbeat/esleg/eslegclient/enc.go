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

package eslegclient

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

type BodyEncoder interface {
	bulkBodyEncoder
	Reader() io.Reader
	Marshal(doc interface{}) error
}

type bulkBodyEncoder interface {
	BulkWriter

	AddHeader(*http.Header)
	Reset()
}

type BulkWriter interface {
	Add(meta, obj interface{}) error
	AddRaw(raw interface{}) error
}

type jsonEncoder struct {
	buf    *bytes.Buffer
	folder *gotype.Iterator

	escapeHTML bool
}

type gzipEncoder struct {
	buf    *bytes.Buffer
	gzip   *gzip.Writer
	folder *gotype.Iterator

	escapeHTML bool
}

type event struct {
	Timestamp time.Time `struct:"@timestamp"`
	Fields    mapstr.M  `struct:",inline"`
}

func NewJSONEncoder(buf *bytes.Buffer, escapeHTML bool) *jsonEncoder {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	e := &jsonEncoder{buf: buf, escapeHTML: escapeHTML}
	e.resetState()
	return e
}

func (b *jsonEncoder) Reset() {
	b.buf.Reset()
}

func (b *jsonEncoder) resetState() {
	var err error
	visitor := json.NewVisitor(b.buf)
	visitor.SetEscapeHTML(b.escapeHTML)

	b.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder()))
	if err != nil {
		panic(err)
	}
}

func (b *jsonEncoder) AddHeader(header *http.Header) {
	header.Add("Content-Type", "application/json; charset=UTF-8")
}

func (b *jsonEncoder) Reader() io.Reader {
	return b.buf
}

func (b *jsonEncoder) Marshal(obj interface{}) error {
	b.Reset()
	return b.AddRaw(obj)
}

// RawEncoding is used to wrap objects that have already been json-encoded,
// so the encoder knows to append them directly instead of treating them
// like a string.
type RawEncoding struct {
	Encoding []byte
}

func (b *jsonEncoder) AddRaw(obj interface{}) error {
	var err error
	switch v := obj.(type) {
	case beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case *beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case RawEncoding:
		_, err = b.buf.Write(v.Encoding)
	default:
		err = b.folder.Fold(obj)
	}

	if err != nil {
		b.resetState()
	}

	b.buf.WriteByte('\n')

	return err
}

func (b *jsonEncoder) Add(meta, obj interface{}) error {
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

func NewGzipEncoder(level int, buf *bytes.Buffer, escapeHTML bool) (*gzipEncoder, error) {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	w, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, err
	}

	g := &gzipEncoder{buf: buf, gzip: w, escapeHTML: escapeHTML}
	g.resetState()
	return g, nil
}

func (g *gzipEncoder) resetState() {
	var err error
	visitor := json.NewVisitor(g.gzip)
	visitor.SetEscapeHTML(g.escapeHTML)

	g.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder()))
	if err != nil {
		panic(err)
	}
}

func (g *gzipEncoder) Reset() {
	g.buf.Reset()
	g.gzip.Reset(g.buf)
}

func (g *gzipEncoder) Reader() io.Reader {
	g.gzip.Close()
	return g.buf
}

func (g *gzipEncoder) AddHeader(header *http.Header) {
	header.Add("Content-Type", "application/json; charset=UTF-8")
	header.Add("Content-Encoding", "gzip")
}

func (g *gzipEncoder) Marshal(obj interface{}) error {
	g.Reset()
	return g.AddRaw(obj)
}

var nl = []byte("\n")

func (g *gzipEncoder) AddRaw(obj interface{}) error {
	var err error
	switch v := obj.(type) {
	case beat.Event:
		err = g.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case *beat.Event:
		err = g.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case RawEncoding:
		_, err = g.gzip.Write(v.Encoding)
	default:
		err = g.folder.Fold(obj)
	}

	if err != nil {
		g.resetState()
	}

	_, err = g.gzip.Write(nl)
	if err != nil {
		g.resetState()
	}

	return nil
}

func (g *gzipEncoder) Add(meta, obj interface{}) error {
	pos := g.buf.Len()
	if err := g.AddRaw(meta); err != nil {
		g.buf.Truncate(pos)
		return err
	}
	if err := g.AddRaw(obj); err != nil {
		g.buf.Truncate(pos)
		return err
	}

	g.gzip.Flush()
	return nil
}
