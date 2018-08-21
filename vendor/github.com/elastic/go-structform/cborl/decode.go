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

package cborl

import (
	"io"

	structform "github.com/elastic/go-structform"
)

type Decoder struct {
	p Parser

	buffer  []byte
	buffer0 []byte
	in      io.Reader
}

func NewDecoder(in io.Reader, buffer int, vs structform.Visitor) *Decoder {
	dec := &Decoder{
		buffer0: make([]byte, buffer),
		in:      in,
	}
	dec.p.init(vs)
	return dec
}

func NewBytesDecoder(b []byte, vs structform.Visitor) *Decoder {
	dec := &Decoder{
		buffer:  b,
		buffer0: b[:0],
		in:      nil,
	}
	dec.p.init(vs)
	return dec
}

func (dec *Decoder) Next() error {
	var (
		n        int
		err      error
		reported bool
	)

	for !reported {
		if len(dec.buffer) == 0 {
			if dec.in == nil {
				return io.EOF
			}

			n, err := dec.in.Read(dec.buffer0)
			dec.buffer = dec.buffer0[:n]
			if err != nil {
				return err
			}
		}

		n, reported, err = dec.p.feedUntil(dec.buffer)
		if err != nil {
			return err
		}

		dec.buffer = dec.buffer[n:]
		if reported {
			return nil
		}
	}

	return nil
}
