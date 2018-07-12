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

package line

import (
	"fmt"
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
)

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type Reader struct {
	lineScanner *lineScanner
}

// New creates a new reader object
func New(input io.Reader, codec encoding.Encoding, separator []byte, bufferSize int) *Reader {
	decReader := newDecoderReader(input, codec, bufferSize)
	lineScanner := newLineScanner(decReader, separator, bufferSize)

	return &Reader{
		lineScanner: lineScanner,
	}
}

// Next reads the next line until the new line character
func (r *Reader) Next() ([]byte, int, error) {
	return r.lineScanner.scan()
}

type decoderReader struct {
	in         io.Reader
	decoder    transform.Transformer
	buf        *streambuf.Buffer
	encodedBuf *streambuf.Buffer
	bufferSize int
	symlen     []int

	offset      int
	bytesOffset int
}

// GetState returns the state of this and the previous readers
func (r *Reader) GetState() common.MapStr {
	return common.MapStr{
		"decoder": common.MapStr{
			"offset": r.lineScanner.in.offset,
			"bytes":  r.lineScanner.in.bytesOffset,
		},
		"scanner": common.MapStr{
			"offset": r.lineScanner.offset,
			"bytes":  r.lineScanner.bytesOffset,
		},
	}
}

func newDecoderReader(in io.Reader, codec encoding.Encoding, bufferSize int) *decoderReader {
	return &decoderReader{
		in:          in,
		decoder:     codec.NewDecoder(),
		buf:         streambuf.New(nil),
		encodedBuf:  streambuf.New(nil),
		bufferSize:  bufferSize,
		offset:      0,
		bytesOffset: 0,
	}
}

func (r *decoderReader) read(buf []byte) (int, error) {
	b := make([]byte, r.bufferSize)

	if r.buf.Len() != 0 {
		return r.copyToOut(buf)
	}

	for {
		start := r.encodedBuf.Len()
		n, err := r.in.Read(b[start:])
		if err != nil {
			return len(buf[:n]), err
		}

		if start > 0 {
			enc, _ := r.encodedBuf.Collect(start)
			b = append(enc, b[start:]...)
		}

		nBytes, nProcessed, err := r.conv(b[:start+n], buf)
		if err != nil {
			if err == transform.ErrShortSrc {
				r.encodedBuf.Append(b[nProcessed:])

				r.offset += nBytes
				r.bytesOffset += nProcessed
				return nBytes, nil
			}
			return 0, err
		}
		r.offset += nBytes
		r.bytesOffset += nProcessed

		return nBytes, nil
	}
}

// msgSize returns the size of the encoded message on the disk
func (r *decoderReader) msgSize(symlen []int, size int) (int, []int, error) {
	fmt.Printf("%+v\n", symlen, size)

	n := 0
	for size > 0 {
		if len(symlen) <= n {
			return 0, symlen, fmt.Errorf("error calculating size: too short symlen")
		}

		size -= symlen[n]
		n++
	}

	symlen = symlen[n:]

	return n, symlen, nil
}

func (r *decoderReader) symbolsLen() []int {
	s := r.symlen
	r.symlen = []int{}
	return s
}

// conv converts encoded bytes into UTF-8 and produces a symlen array which
// records the size of the encoded bytes and its converted size
func (r *decoderReader) conv(in []byte, out []byte) (int, int, error) {
	var err error
	nProcessed := 0
	decodedChar := make([]byte, 64)
	r.symlen = make([]int, len(in))

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

			r.symlen[nProcessed] = nDst
			r.buf.Write(decodedChar[:nDst])
			nProcessed++
			break
		}
		i = j
	}

	n, err := r.copyToOut(out)
	fmt.Println(r.symlen)
	return n, nProcessed, err
}

func (r *decoderReader) copyToOut(out []byte) (int, error) {
	until := len(out)
	if r.buf.Len() < until {
		until = r.buf.Len()
	}
	b, err := r.buf.Collect(until)
	return copy(out, b), err
}

type lineScanner struct {
	in         *decoderReader
	separator  []byte
	bufferSize int

	symlen      []int
	buf         *streambuf.Buffer
	offset      int
	bytesOffset int
}

func newLineScanner(in *decoderReader, separator []byte, bufferSize int) *lineScanner {
	return &lineScanner{
		in:          in,
		separator:   separator,
		bufferSize:  bufferSize,
		buf:         streambuf.New(nil),
		offset:      0,
		bytesOffset: 0,
		symlen:      []int{},
	}
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.Index(s.separator)
	for !separatorFound(idx) {
		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		if err != nil {
			return nil, 0, err
		}

		s.buf.Append(b[:n])
		s.symlen = append(s.symlen, s.in.symbolsLen()...)
		idx = s.buf.Index(s.separator)
	}

	return s.line(idx)
}

// separatorFound checks if a new separator was found.
func separatorFound(i int) bool {
	return i != -1
}

// line sets the offset of the scanner and returns a line.
func (s *lineScanner) line(i int) ([]byte, int, error) {
	line, err := s.buf.CollectUntil(s.separator)
	if err != nil {
		panic(err)
	}

	var msgSymbols int
	msgSymbols, s.symlen, err = s.in.msgSize(s.symlen, len(line))
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("%+v\n", s.symlen)
	s.bytesOffset += msgSymbols
	s.offset += i
	s.buf.Reset()

	return line, len(line), nil
}
