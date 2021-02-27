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
	"compress/flate"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

var (
	decoders = map[string]func(io.Reader) (io.ReadCloser, error){
		"gzip":      decodeGZIP,
		"x-gzip":    decodeGZIP,
		"deflate":   decodeDeflate,
		"x-deflate": decodeDeflate,

		// Not really expected, withdrawn by RFC
		"identity": decodeIdentity,

		// Couldn't find an implementation of `compress` nor a server/library
		// that supports it. Seems long dead.
		// "compress":   nil,
		// "x-compress": nil,
	}

	// ErrNoDecoder is returned when an unknown content-encoding is used.
	ErrNoDecoder = errors.New("decoder not found")

	// ErrSizeLimited is returned when
	ErrSizeLimited = errors.New("body truncated due to size limitation")
)

func decodeHTTPBody(data []byte, format string, maxSize int) ([]byte, error) {
	decoder, found := decoders[format]
	if !found {
		return nil, ErrNoDecoder
	}
	reader, err := decoder(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return readMax(reader, maxSize)
}

func decodeGZIP(reader io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(reader)
}

func decodeDeflate(reader io.Reader) (io.ReadCloser, error) {
	return flate.NewReader(reader), nil
}

type closeDecorator struct {
	io.Reader
}

func (closeDecorator) Close() error {
	return nil
}

func decodeIdentity(reader io.Reader) (io.ReadCloser, error) {
	return closeDecorator{reader}, nil
}

func readMax(reader io.Reader, maxSize int) (result []byte, err error) {
	const minSize = 512
	for used := 0; ; {
		if len(result)-used < minSize {
			grow := len(result) >> 1
			if grow < minSize {
				grow = minSize
			}
			result = append(result, make([]byte, grow)...)
		}
		n, err := reader.Read(result[used:])
		if n > 0 {
			used += n
			if used > maxSize {
				used = maxSize
				err = ErrSizeLimited
			}
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return result[:used], err
		}
	}
}
