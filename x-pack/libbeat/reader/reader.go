// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
)

// Moved from x-pack/filebeat/input/awss3/s3_objects.go
//
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

// Moved from x-pack/filebeat/input/awss3/s3_object.go
//
// addGzipDecoderIfNeeded determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not and adds gzipped decoder if needed. A buffered reader is used
// so the function can peek into the byte  stream without consuming it. This makes it convenient for
// code executed after this function call to consume the stream if it wants.
func AddGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := IsStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if !gzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}
