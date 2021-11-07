// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package logs

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
)

// chunkEncoder implements log buffer chunking and compression. Log events are
// written to the encoder and the encoder outputs chunks that are fit to the
// configured limit.
type chunkEncoder struct {
	limit        int64
	bytesWritten int
	buf          *bytes.Buffer
	w            *gzip.Writer
}

func newChunkEncoder(limit int64) *chunkEncoder {
	enc := &chunkEncoder{
		limit: limit,
	}
	enc.reset()

	return enc
}

func (enc *chunkEncoder) Write(event EventV1) (result []byte, err error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(event); err != nil {
		return nil, err
	}

	bs := buf.Bytes()

	if len(bs) == 0 {
		return nil, nil
	} else if int64(len(bs)+2) > enc.limit {
		return nil, fmt.Errorf("upload chunk size too small")
	}

	if int64(len(bs)+enc.bytesWritten+1) > enc.limit {
		if err := enc.writeClose(); err != nil {
			return nil, err
		}

		result = enc.reset()
	}

	if enc.bytesWritten == 0 {
		n, err := enc.w.Write([]byte(`[`))
		if err != nil {
			return nil, err
		}
		enc.bytesWritten += n
	} else {
		n, err := enc.w.Write([]byte(`,`))
		if err != nil {
			return nil, err
		}
		enc.bytesWritten += n
	}

	n, err := enc.w.Write(bs)
	if err != nil {
		return nil, err
	}

	enc.bytesWritten += n
	return
}

func (enc *chunkEncoder) writeClose() error {
	if _, err := enc.w.Write([]byte(`]`)); err != nil {
		return err
	}
	return enc.w.Close()
}

func (enc *chunkEncoder) Flush() ([]byte, error) {
	if enc.bytesWritten == 0 {
		return nil, nil
	}
	if err := enc.writeClose(); err != nil {
		return nil, err
	}
	return enc.reset(), nil
}

func (enc *chunkEncoder) reset() []byte {
	buf := enc.buf
	enc.buf = new(bytes.Buffer)
	enc.bytesWritten = 0
	enc.w = gzip.NewWriter(enc.buf)
	if buf != nil {
		return buf.Bytes()
	}
	return nil
}

// chunkDecoder decodes the encoded chunks and outputs the log events
type chunkDecoder struct {
	raw []byte
}

func newChunkDecoder(raw []byte) *chunkDecoder {
	return &chunkDecoder{
		raw: raw,
	}
}

func (dec *chunkDecoder) decode() ([]EventV1, error) {
	gr, err := gzip.NewReader(bytes.NewReader(dec.raw))
	if err != nil {
		return nil, err
	}

	var events []EventV1
	if err := json.NewDecoder(gr).Decode(&events); err != nil {
		return nil, err
	}
	gr.Close()

	return events, nil
}
