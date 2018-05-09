package elasticsearch

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"time"

	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

type bodyEncoder interface {
	bulkBodyEncoder
	Reader() io.Reader
	Marshal(doc interface{}) error
}

type bulkBodyEncoder interface {
	bulkWriter

	AddHeader(*http.Header)
	Reset()
}

type bulkWriter interface {
	Add(meta, obj interface{}) error
	AddRaw(raw interface{}) error
}

type jsonEncoder struct {
	buf    *bytes.Buffer
	folder *gotype.Iterator
}

type gzipEncoder struct {
	buf    *bytes.Buffer
	gzip   *gzip.Writer
	folder *gotype.Iterator
}

type event struct {
	Timestamp time.Time     `struct:"@timestamp"`
	Fields    common.MapStr `struct:",inline"`
}

func newJSONEncoder(buf *bytes.Buffer) *jsonEncoder {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	e := &jsonEncoder{buf: buf}
	e.resetState()
	return e
}

func (b *jsonEncoder) Reset() {
	b.buf.Reset()
}

func (b *jsonEncoder) resetState() {
	var err error
	visitor := json.NewVisitor(b.buf)
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

func (b *jsonEncoder) AddRaw(obj interface{}) error {
	var err error
	switch v := obj.(type) {
	case beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case *beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
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

func newGzipEncoder(level int, buf *bytes.Buffer) (*gzipEncoder, error) {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	w, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, err
	}

	g := &gzipEncoder{buf: buf, gzip: w}
	g.resetState()
	return g, nil
}

func (g *gzipEncoder) resetState() {
	var err error
	visitor := json.NewVisitor(g.gzip)
	g.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder()))
	if err != nil {
		panic(err)
	}
}

func (b *gzipEncoder) Reset() {
	b.buf.Reset()
	b.gzip.Reset(b.buf)
}

func (b *gzipEncoder) Reader() io.Reader {
	b.gzip.Close()
	return b.buf
}

func (b *gzipEncoder) AddHeader(header *http.Header) {
	header.Add("Content-Type", "application/json; charset=UTF-8")
	header.Add("Content-Encoding", "gzip")
}

func (b *gzipEncoder) Marshal(obj interface{}) error {
	b.Reset()
	return b.AddRaw(obj)
}

var nl = []byte("\n")

func (b *gzipEncoder) AddRaw(obj interface{}) error {
	var err error
	switch v := obj.(type) {
	case beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	case *beat.Event:
		err = b.folder.Fold(event{Timestamp: v.Timestamp, Fields: v.Fields})
	default:
		err = b.folder.Fold(obj)
	}

	if err != nil {
		b.resetState()
	}

	_, err = b.gzip.Write(nl)
	if err != nil {
		b.resetState()
	}

	return nil
}

func (b *gzipEncoder) Add(meta, obj interface{}) error {
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
