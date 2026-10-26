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

package cassandra

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
)

func TestReadInt(t *testing.T) {
	tests := []struct {
		wire uint32
		want int
	}{
		{wire: 0x00000001, want: 1},
		{wire: 0x7fffffff, want: 2147483647},
		{wire: 0x80000000, want: -2147483648},
		{wire: 0xffffffff, want: -1},
	}
	for _, tc := range tests {
		data := binary.BigEndian.AppendUint32(nil, tc.wire)
		stream := StreamDecoder{r: streambuf.New(data)}
		array := ByteArrayDecoder{Data: &data}

		assert.Equal(t, tc.want, stream.ReadInt(), "StreamDecoder.ReadInt(%#x)", tc.wire)
		assert.Equal(t, tc.want, array.ReadInt(), "ByteArrayDecoder.ReadInt(%#x)", tc.wire)
	}
}

func TestStreamDecoderReadBytesLength(t *testing.T) {
	authSuccess := func(length int32, payload string) []byte {
		return append(appendInt(nil, length), payload...)
	}

	tests := []struct {
		name    string
		body    []byte
		want    map[string]any
		wantErr string
	}{
		{
			name: "valid token",
			body: authSuccess(3, "abc"),
			want: map[string]any{"authentication": map[string]any{"data": `"abc"`}},
		},
		{
			name: "null token",
			body: authSuccess(-1, ""),
			want: map[string]any{"authentication": map[string]any{"data": `""`}},
		},
		{
			name: "other negative length",
			body: authSuccess(-2, ""),
			want: map[string]any{"authentication": map[string]any{"data": `""`}},
		},
		{
			name:    "length beyond buffered body",
			body:    authSuccess(4096, "abc"),
			wantErr: "not enough buf to readBytes, need:4096, actual:3",
		},
		{
			name:    "maximum length",
			body:    authSuccess(0x7fffffff, ""),
			wantErr: "not enough buf to readBytes, need:2147483647, actual:0",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := decodeFrame(t, newFrame(v4Response, 0, opAuthSuccess, tc.body), nil)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "length beyond the body should be rejected before allocating")
				return
			}
			assert.NoError(t, err, "valid frame should parse")
			assert.Equal(t, tc.want, data, "unexpected decoded frame")
		})
	}
}
