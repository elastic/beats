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

package diskqueue

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type nopWriteCloser struct {
	io.Writer
}

func NopWriteCloser(w io.Writer) io.WriteCloser {
	return nopWriteCloser{w}
}
func (nopWriteCloser) Close() error { return nil }

func TestCompressionReader(t *testing.T) {
	tests := map[string]struct {
		plaintext  []byte
		compressed []byte
	}{
		"abc compressed with https://github.com/lz4/lz4 v1.9.3": {
			plaintext: []byte("abc"),
			compressed: []byte{
				0x04, 0x22, 0x4d, 0x18,
				0x64, 0x40, 0xa7, 0x04,
				0x00, 0x00, 0x80, 0x61,
				0x62, 0x63, 0x0a, 0x00,
				0x00, 0x00, 0x00, 0x6c,
				0x3e, 0x7b, 0x08, 0x00},
		},
		"abc compressed with pierrec lz4": {
			plaintext: []byte("abc"),
			compressed: []byte{
				0x04, 0x22, 0x4d, 0x18,
				0x64, 0x70, 0xb9, 0x03,
				0x00, 0x00, 0x80, 0x61,
				0x62, 0x63, 0x00, 0x00,
				0x00, 0x00, 0xff, 0x53,
				0xd1, 0x32},
		},
	}

	for name, tc := range tests {
		dst := make([]byte, len(tc.plaintext))
		src := bytes.NewReader(tc.compressed)
		cr := NewCompressionReader(io.NopCloser(src))
		n, err := cr.Read(dst)
		assert.Nil(t, err, name)
		assert.Equal(t, len(tc.plaintext), n, name)
		assert.Equal(t, tc.plaintext, dst, name)
	}
}

func TestCompressionWriter(t *testing.T) {
	tests := map[string]struct {
		plaintext  []byte
		compressed []byte
	}{
		"abc pierrec lz4": {
			plaintext: []byte("abc"),
			compressed: []byte{
				0x04, 0x22, 0x4d, 0x18,
				0x64, 0x70, 0xb9, 0x03,
				0x00, 0x00, 0x80, 0x61,
				0x62, 0x63, 0x00, 0x00,
				0x00, 0x00, 0xff, 0x53,
				0xd1, 0x32},
		},
	}

	for name, tc := range tests {
		var dst bytes.Buffer
		cw := NewCompressionWriter(NopWriteCloseSyncer(NopWriteCloser(&dst)))
		n, err := cw.Write(tc.plaintext)
		cw.Close()
		assert.Nil(t, err, name)
		assert.Equal(t, len(tc.plaintext), n, name)
		assert.Equal(t, tc.compressed, dst.Bytes(), name)
	}
}

func TestCompressionRoundTrip(t *testing.T) {
	tests := map[string]struct {
		plaintext []byte
	}{
		"no repeat":  {plaintext: []byte("abcdefghijklmnopqrstuvwxzy01234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ")},
		"256 repeat": {plaintext: []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
	}
	for name, tc := range tests {
		pr, pw := io.Pipe()
		src := bytes.NewReader(tc.plaintext)
		var dst bytes.Buffer

		go func() {
			cw := NewCompressionWriter(NopWriteCloseSyncer(pw))
			_, err := io.Copy(cw, src)
			assert.Nil(t, err, name)
			cw.Close()
		}()

		cr := NewCompressionReader(pr)
		_, err := io.Copy(&dst, cr)
		assert.Nil(t, err, name)
		assert.Equal(t, tc.plaintext, dst.Bytes(), name)
	}
}

func TestCompressionSync(t *testing.T) {
	tests := map[string]struct {
		plaintext []byte
	}{
		"no repeat":  {plaintext: []byte("abcdefghijklmnopqrstuvwxzy01234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ")},
		"256 repeat": {plaintext: []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
	}
	for name, tc := range tests {
		pr, pw := io.Pipe()
		var dst bytes.Buffer
		go func() {
			cw := NewCompressionWriter(NopWriteCloseSyncer(pw))
			src1 := bytes.NewReader(tc.plaintext)
			_, err := io.Copy(cw, src1)
			assert.Nil(t, err, name)
			//prior to v4.1.15 of pierrec/lz4 there was a
			// bug that prevented writing after a Flush.
			// The call to Sync here exercises Flush.
			err = cw.Sync()
			assert.Nil(t, err, name)
			src2 := bytes.NewReader(tc.plaintext)
			_, err = io.Copy(cw, src2)
			assert.Nil(t, err, name)
			cw.Close()
		}()
		cr := NewCompressionReader(pr)
		_, err := io.Copy(&dst, cr)
		assert.Nil(t, err, name)
		assert.Equal(t, tc.plaintext, dst.Bytes()[:len(tc.plaintext)], name)
		assert.Equal(t, tc.plaintext, dst.Bytes()[len(tc.plaintext):], name)
	}
}
