package elasticsearch

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
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
	buf *bytes.Buffer
}

type gzipEncoder struct {
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

func (b *jsonEncoder) AddHeader(header *http.Header) {
	header.Add("Content-Type", "application/json; charset=UTF-8")
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

func (b *gzipEncoder) AddHeader(header *http.Header) {
	header.Add("Content-Type", "application/json; charset=UTF-8")
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
