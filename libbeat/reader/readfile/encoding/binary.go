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

package encoding

import (
	g_binary "encoding/binary"
	"fmt"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type BinaryEncodingConfig struct {
	offset int
	length int
	order  g_binary.ByteOrder
}

type binenc struct {}

// func (b binary) Reset() {}
// func (b binary) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) { }

func (binenc) NewDecoder() *encoding.Decoder {
	fmt.Println("Creating new Binary Decoder")
	return &encoding.Decoder{
		Transformer: transform.Nop,
	}
}

func (binenc) NewEncoder() *encoding.Encoder {
	fmt.Println("Creating new Binary Encoder")
	return &encoding.Encoder{
		Transformer: transform.Nop,
	}
}
