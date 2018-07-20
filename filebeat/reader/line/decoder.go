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
	"io"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/filebeat/reader/encode/encoding"
	"github.com/elastic/beats/filebeat/reader/line/converter"
	"github.com/elastic/beats/libbeat/common/streambuf"
)

type decoderReader struct {
	in         io.Reader
	converter  converter.Converter
	encodedBuf *streambuf.Buffer
	bufferSize int
}

func newDecoderReader(in io.Reader, codec encoding.Encoding, name string, bufferSize int) (*decoderReader, error) {
	f, err := converter.GetFactory(name)
	if err != nil {
		return nil, err
	}

	c, err := f(codec.NewDecoder(), bufferSize)
	if err != nil {
		return nil, err
	}

	return &decoderReader{
		in:         in,
		converter:  c,
		encodedBuf: streambuf.New(nil),
		bufferSize: bufferSize,
	}, nil
}

func (r *decoderReader) read(buf []byte) (int, error) {
	b := make([]byte, r.bufferSize)

	n, err := r.converter.Collect(buf)
	if n != 0 {
		return n, err
	}

	for {
		start := r.encodedBuf.Len()
		n, err := r.in.Read(b[start:])
		if err != nil {
			return len(buf[:n]), err
		}

		if start > 0 {
			enc, err := r.encodedBuf.Collect(start)
			if err != nil {
				return 0, err
			}
			r.encodedBuf.Reset()
			b = append(enc, b[start:]...)
		}

		nBytes, nProcessed, err := r.converter.Convert(b[:start+n], buf)
		if err != nil {
			if err == transform.ErrShortSrc {
				r.encodedBuf.Append(b[nProcessed:])
				return nBytes, nil
			}
			return 0, err
		}
		return nBytes, nil
	}
}

func (r *decoderReader) msgSize(symlen []uint8, size int) (int, []uint8, error) {
	return r.converter.MsgSize(symlen, size)
}

func (r *decoderReader) GetSymLen() []uint8 {
	return r.converter.GetSymLen()
}
