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
	"fmt"
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
	bufferSize int
	symlen     []uint8
}

func newDecoderReader(in io.Reader, codec encoding.Encoding, bufferSize int) *decoderReader {
	return &decoderReader{
		in:         in,
		decoder:    codec.NewDecoder(),
		buf:        streambuf.New(nil),
		encodedBuf: streambuf.New(nil),
		bufferSize: bufferSize,
	}
}

func (r *decoderReader) read(buf []byte) (int, error) {
	// TODO add to the beginning of buf and then start decoding
	// instead of returning
	if r.buf.Len() != 0 {
		return r.copyToOut(buf)
	}

	b := make([]byte, r.bufferSize)
	start := r.encodedBuf.Len()
	n, err := r.in.Read(b[start:])
	if err != nil {
		return 0, err
	}

	if start > 0 {
		enc, err := r.encodedBuf.Collect(start)
		if err != nil {
			return 0, err
		}
		r.encodedBuf.Reset()
		b = append(enc, b[start:]...)
	}

	nBytes, nProcessed, err := r.conv(b[:start+n], buf)
	if err != nil {
		if err == transform.ErrShortSrc {
			r.encodedBuf.Append(b[nProcessed:])
			return nBytes, nil
		}
		return 0, err
	}
	return nBytes, nil
}

// msgSize returns the size of the encoded message on the disk
// sizeInUTF8 is the lenght of the converted UTF-8 line
func (r *decoderReader) msgSize(symlen []uint8, sizeInUTF8 uint) (int, []uint8, error) {
	nEncodedBytes := 0
	for sizeInUTF8 > 0 {
		if len(symlen) <= nEncodedBytes {
			return 0, symlen, fmt.Errorf("error calculating size: too short symlen")
		}

		sizeInUTF8 -= uint(symlen[nEncodedBytes])
		nEncodedBytes++
	}

	symlen = symlen[nEncodedBytes:]

	return nEncodedBytes, symlen, nil
}

func (r *decoderReader) symbolsLen() []uint8 {
	s := r.symlen
	r.symlen = []uint8{}
	return s
}

// conv converts encoded bytes into UTF-8 and produces a symlen array which
// records the size of the encoded bytes and its converted size
func (r *decoderReader) conv(in []byte, out []byte) (int, int, error) {
	var err error
	nProcessed := 0
	decodedChar := make([]byte, 64)
	r.symlen = make([]uint8, len(in))

	i := 0
	srcLen := len(in)
	for i < srcLen {
		j := i + 1

		for j <= srcLen {
			nDst, nSrc, err := r.decoder.Transform(decodedChar, in[i:j], false)
			if err != nil {
				// if no char is decoded, try increasing the input buffer
				if err == transform.ErrShortSrc {
					j++

					// if the buffer size cannot be increased, return what's been decoded and an error
					if srcLen < j {
						n, _ := r.copyToOut(out)
						r.symlen = r.symlen[:nProcessed]
						return n, nProcessed, err
					}
				}
				err = nil
			}

			// move in the symlen buffer if no char is decoded
			if nDst == 0 && nSrc == 0 {
				nProcessed++
				continue
			}

			r.symlen[nProcessed] = uint8(nDst)
			r.buf.Write(decodedChar[:nDst])
			nProcessed++
			break
		}
		i = j
	}

	n, err := r.copyToOut(out)
	return n, nProcessed, err
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
