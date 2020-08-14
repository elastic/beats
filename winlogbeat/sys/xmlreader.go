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

package sys

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// The type xmlSafeReader escapes UTF control characters in the io.Reader
// it wraps, so that it can be fed to Go's xml parser.
// Characters for which `unicode.IsControl` returns true will be output as
// an hexadecimal unicode escape sequence "\\uNNNN".
type xmlSafeReader struct {
	inner   io.Reader
	backing [256]byte
	buf     []byte
	code    []byte
}

var _ io.Reader = (*xmlSafeReader)(nil)

func output(n int) (int, error) {
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

// Read implements the io.Reader interface.
func (r *xmlSafeReader) Read(d []byte) (n int, err error) {
	if len(r.code) > 0 {
		n = copy(d, r.code)
		r.code = r.code[n:]
		return output(n)
	}
	if len(r.buf) == 0 {
		n, _ = r.inner.Read(r.backing[:])
		r.buf = r.backing[:n]
	}
	for i := 0; i < len(r.buf); {
		code, size := utf8.DecodeRune(r.buf[i:])
		if !unicode.IsSpace(code) && unicode.IsControl(code) {
			n = copy(d, r.buf[:i])
			r.buf = r.buf[n+1:]
			r.code = []byte(fmt.Sprintf("\\u%04x", code))
			m := copy(d[n:], r.code)
			r.code = r.code[m:]
			return output(n + m)
		}
		i += size
	}
	n = copy(d, r.buf)
	r.buf = r.buf[n:]
	return output(n)
}

func newXMLSafeReader(rawXML []byte) io.Reader {
	return &xmlSafeReader{inner: bytes.NewReader(rawXML)}
}
