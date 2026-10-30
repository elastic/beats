// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoding

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// addGzipDecoderIfNeeded determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not and adds gzipped decoder if needed. A bufio.Reader is used
// so the function can peek into the byte  stream without consuming it. This makes it convenient for
// code executed after this function call to consume the stream if it wants.
func addGzipDecoderIfNeeded(reader *bufio.Reader) (io.Reader, error) {
	isStreamGzipped := false
	// check if stream is gziped or not
	buf, err := reader.Peek(3)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = nil
		}
		return reader, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	isStreamGzipped = bytes.Equal(buf, []byte{0x1F, 0x8B, 0x08})

	if !isStreamGzipped {
		return reader, nil
	}

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return gzReader, nil
}

// evaluateJSON, uses a bufio.NewReader & reader.Peek to evaluate if the
// data stream contains a json array as the root element or not, without
// advancing the reader. If the data stream contains an array as the root
// element, the value of the boolean return type is set to true.
func evaluateJSON(reader *bufio.Reader) (io.Reader, bool, error) {
	eof := false
	for i := 0; ; i++ {
		b, err := reader.Peek((i + 1) * 5)
		if errors.Is(err, io.EOF) {
			eof = true
		}
		startByte := i * 5
		for j := 0; j < len(b[startByte:]); j++ {
			char := b[startByte+j : startByte+j+1]
			switch {
			case bytes.Equal(char, []byte("[")):
				return reader, true, nil
			case bytes.Equal(char, []byte("{")):
				return reader, false, nil
			case unicode.IsSpace(bytes.Runes(char)[0]):
				continue
			default:
				return nil, false, fmt.Errorf("unexpected error: JSON data is malformed")
			}
		}
		if eof {
			return nil, false, fmt.Errorf("unexpected error: JSON data is malformed")
		}
	}
}
