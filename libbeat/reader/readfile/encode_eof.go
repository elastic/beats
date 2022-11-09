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

package readfile

import (
	"bytes"
	"io"
	"time"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// EncoderReaderEof produces lines by reading until EOF from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type EncoderReaderEof struct {
	reader     io.ReadCloser
	maxBytes   int // max bytes per line limit to avoid OOM with malformatted files
	inBuffer   *streambuf.Buffer
	outBuffer  *streambuf.Buffer
	inOffset   int // input buffer read offset
	byteCount  int // number of bytes decoded from input buffer into output buffer
	decoder    transform.Transformer
	tempBuffer []byte
	logger     *logp.Logger
}

// NewEncodeReaderEof creates a new Encode reader from input reader by applying
// the given codec.
func NewEncodeReaderEof(r io.ReadCloser, config Config) *EncoderReaderEof {
	return &EncoderReaderEof{
		reader:     r,
		maxBytes:   config.MaxBytes,
		decoder:    config.Codec.NewDecoder(),
		inBuffer:   streambuf.New(nil),
		outBuffer:  streambuf.New(nil),
		tempBuffer: make([]byte, config.BufferSize),
		logger:     logp.NewLogger("reader_eof"),
	}
}

// Next reads until EOF from its initial io.Reader
// This converts a io.Reader to a reader.reader
func (r *EncoderReaderEof) Next() (reader.Message, error) {
	idx := 0
	var err error
	for {
		// Try to read more bytes into buffer
		n := 0
		n, err = r.reader.Read(r.tempBuffer)
		idx += n

		if err == io.EOF && n > 0 {
			// Continue processing the returned bytes. The next call will yield EOF with 0 bytes.
			err = nil
		}

		if err != nil {
			break
		}

		// Write to buffer also in case of err
		_, _ = r.inBuffer.Write(r.tempBuffer[:n])

		// If max bytes limit is set, returns
		if r.maxBytes != 0 && idx > r.maxBytes {
			r.logger.Warnf("Exceeded %d max bytes", r.maxBytes)
			break
		} else if r.inBuffer.Len() > r.maxBytes {
			r.logger.Warnf("Exceeded %d max bytes", r.maxBytes)
			break
		}
	}

	// Found encoded byte sequence for newline in buffer
	// -> decode input sequence into outBuffer
	sz, decodeErr := r.decode(idx)
	if decodeErr != nil {
		r.logger.Errorf("Error decoding line: %s", err)
		// In case of error increase size by unencoded length
		return reader.Message{}, decodeErr
	}

	// output buffer contains until EOF or max_bytes. Extract
	// byte slice from buffer and reset output buffer.
	collectedBytes, collectErr := r.outBuffer.Collect(r.outBuffer.Len())
	r.outBuffer.Reset()
	if collectErr != nil {
		// This should never happen as otherwise we have a broken state
		panic(collectErr)
	}

	// Creating message object
	return reader.Message{
		Ts:      time.Now(),
		Content: bytes.Replace(collectedBytes, []byte("\xef\xbb\xbf"), []byte(""), -1),
		Bytes:   sz,
		Fields:  mapstr.M{},
	}, err
}

func (r *EncoderReaderEof) decode(end int) (int, error) {
	var err error
	inBytes := r.inBuffer.Bytes()
	start := 0

	for start < end {
		var nDst, nSrc int

		nDst, nSrc, err = r.decoder.Transform(r.tempBuffer, inBytes[start:end], false)
		if err != nil {
			// Check if error is different from destination buffer too short
			if err != transform.ErrShortDst {
				r.outBuffer.Write(inBytes[0:end])
				start = end
				break
			}

			// Reset error as decoding continues
			err = nil
		}

		start += nSrc
		r.outBuffer.Write(r.tempBuffer[:nDst])
	}

	r.byteCount += start
	return start, err
}

func (r *EncoderReaderEof) Close() error {
	return r.reader.Close()
}
