// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"

	"github.com/klauspost/compress/snappy"
)

// snappyStreamID is the 10-byte stream identifier chunk that begins every
// Snappy framed stream. See https://github.com/google/snappy/blob/main/framing_format.txt.
var snappyStreamID = []byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59}

// IsStreamGzipped determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not. A buffered reader is used so the function can peek into the byte
// stream without consuming it. This makes it convenient for code executed after this function call
// to consume the stream if it wants.
func IsStreamGzipped(r *bufio.Reader) (bool, error) {
	buf, err := r.Peek(3)
	if err != nil && err != io.EOF {
		return false, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	return bytes.HasPrefix(buf, []byte{0x1F, 0x8B, 0x08}), nil
}

// IsStreamSnappy determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents Snappy framed content or not. A buffered reader is used so the function can peek into
// the byte stream without consuming it.
func IsStreamSnappy(r *bufio.Reader) (bool, error) {
	buf, err := r.Peek(len(snappyStreamID))
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	return bytes.HasPrefix(buf, snappyStreamID), nil
}

// AddDecoderIfNeeded determines whether the given stream of bytes represents
// gzipped or Snappy framed content and adds the appropriate decoder if needed.
// A buffered reader is used so the function can peek into the byte stream
// without consuming it. This makes it convenient for code executed after this
// function call to consume the stream if it wants.
func AddDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := IsStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if gzipped {
		return gzip.NewReader(bufReader)
	}

	snapped, err := IsStreamSnappy(bufReader)
	if err != nil {
		return nil, err
	}
	if snapped {
		return snappy.NewReader(bufReader), nil
	}

	return bufReader, nil
}
