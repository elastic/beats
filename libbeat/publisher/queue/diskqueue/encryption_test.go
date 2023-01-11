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
	"crypto/aes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NopWriteCloseSyncer(w io.WriteCloser) WriteCloseSyncer {
	return nopWriteCloseSyncer{w}
}

type nopWriteCloseSyncer struct {
	io.WriteCloser
}

func (nopWriteCloseSyncer) Sync() error { return nil }

func TestEncryptionRoundTrip(t *testing.T) {
	tests := map[string]struct {
		plaintext []byte
	}{
		"8 bits":   {plaintext: []byte("a")},
		"128 bits": {plaintext: []byte("bbbbbbbbbbbbbbbb")},
		"136 bits": {plaintext: []byte("ccccccccccccccccc")},
	}
	for name, tc := range tests {
		pr, pw := io.Pipe()
		src := bytes.NewReader(tc.plaintext)
		var dst bytes.Buffer
		var teeBuf bytes.Buffer
		key := []byte("kkkkkkkkkkkkkkkk")

		go func() {
			//NewEncryptionWriter writes iv, so needs to be in go routine
			ew, err := NewEncryptionWriter(NopWriteCloseSyncer(pw), key)
			assert.Nil(t, err, name)
			_, err = io.Copy(ew, src)
			assert.Nil(t, err, name)
			ew.Close()
		}()

		tr := io.TeeReader(pr, &teeBuf)
		er, err := NewEncryptionReader(io.NopCloser(tr), key)
		assert.Nil(t, err, name)
		_, err = io.Copy(&dst, er)
		assert.Nil(t, err, name)
		// Check round trip worked
		assert.Equal(t, tc.plaintext, dst.Bytes(), name)
		// Check that iv & cipher text were written
		assert.Equal(t, len(tc.plaintext)+aes.BlockSize, teeBuf.Len(), name)
		// Check that cipher text and plaintext don't match
		assert.NotEqual(t, tc.plaintext, teeBuf.Bytes()[aes.BlockSize:], name)
	}
}
