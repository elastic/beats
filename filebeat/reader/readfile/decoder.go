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
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

// decoderReader reads from an input file and converts its content into UTF-8.
// It stores the size of the original bytes, so offset can be calculated later in
// the line scanner.
type decoderReader struct {
	in         io.Reader
	decoder    transform.Transformer
	buf        *streambuf.Buffer
	encodedBuf *streambuf.Buffer
	offset     int // segment byte offset

	bufferSize int
}

func newDecoderReader(in io.Reader, codec encoding.Encoding, bufferSize int) (*decoderReader, error) {
	r := &decoderReader{
		in:         in,
		decoder:    codec.NewDecoder(),
		buf:        streambuf.New(nil),
		encodedBuf: streambuf.New(nil),
		offset:     0, // TODO init correctly
		bufferSize: bufferSize,
	}

	return r, nil
}

// seekToLastRead reads from the input file until the saved offset is reached.
// It is the substitution of seeking in the file.
// This function is no longer required after registry refactoring is done.
func (r *decoderReader) seekToLastRead(convertedOffset int) error {
	// TODO move seeking in encoded stream to harvester `file.Seek`
	err := r.seekInReader(r.offset, r.in)
	if err != nil {
		return err
	}

	return r.seekInReader(convertedOffset, r)
}

// this is a temp function to simplify seekToLastRead
func (r *decoderReader) seekInReader(offset int, reader io.Reader) error {
	if offset == 0 {
		return nil
	}

	b := make([]byte, r.bufferSize)
	remaining := offset
	for remaining > 0 {
		s := r.bufferSize
		if remaining < s {
			s = remaining
		}

		n, err := reader.Read(b[:s])
		remaining -= n
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}

	return nil
}

// read reads from an underlying file and converts it into UTF-8.
func (r *decoderReader) Read(buf []byte) (int, error) {
	// collect leftover converted bytes
	nb := 0
	var err error
	if r.buf.Len() != 0 {
		nb, err = r.copyToOut(buf)
		if err != nil {
			return 0, err
		}
	}

	b := make([]byte, r.bufferSize)
	start := r.encodedBuf.Len()
	// read from the underlying file
	n, err := r.in.Read(b[start:])
	if err != nil {
		if err == io.EOF && start == 0 {
			return 0, err
		}
	}

	// collect leftover encoded bytes
	if start > 0 {
		enc, err := r.encodedBuf.Collect(start)
		if err != nil {
			return 0, err
		}
		r.encodedBuf.Reset()
		b = append(enc, b[start:]...)
	}

	// convert encoded bytes into UTF-8
	nDst, nSrc, err := r.decoder.Transform(buf[nb:], b[:start+n], false)
	if err != nil {
		if err == transform.ErrShortSrc {
			r.encodedBuf.Append(b[nSrc:])
			r.offset += nSrc
			return nDst, nil
		}
		if err == transform.ErrShortDst {
			r.encodedBuf.Append(b[nSrc : start+n])
			r.offset += nSrc
			return nDst, nil
		}
		return 0, err
	}

	r.offset += nSrc
	return nDst, nil
}

func (r *decoderReader) copyToOut(out []byte) (int, error) {
	until := len(out)
	if r.buf.Len() < until {
		until = r.buf.Len()
	}
	b, err := r.buf.Collect(until)
	if err != nil {
		return 0, err
	}
	r.buf.Reset()
	return copy(out, b), nil
}
