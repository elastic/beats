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

	"github.com/golang/snappy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
)

const (
	v4Request  = 0x04
	v4Response = 0x84
)

func newFrame(version, flags byte, op FrameOp, body []byte) []byte {
	frame := []byte{version, flags, 0x00, 0x00, byte(op)}
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(body))) //nolint:gosec // test bodies are tiny
	return append(frame, body...)
}

func decodeFrame(t *testing.T, frame []byte, compressor Compressor) (map[string]any, error) {
	t.Helper()
	framer := NewFramer(streambuf.New(frame), compressor)
	_, err := framer.ReadHeader()
	require.NoError(t, err, "header should parse")
	return framer.ReadFrame()
}

func appendInt32(b []byte, n int32) []byte {
	return binary.BigEndian.AppendUint32(b, uint32(n)) //nolint:gosec // test values keep their bit pattern
}

func TestReadFrameLongStringLength(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    map[string]any
		wantErr string
	}{
		{
			name: "valid query",
			body: append(appendInt32(nil, 8), "SELECT 1"...),
			want: map[string]any{"query": "SELECT 1"},
		},
		{
			name:    "negative length",
			body:    appendInt32(nil, -1),
			wantErr: "invalid long string length: -1",
		},
	}
	decoders := []struct {
		name       string
		compressor Compressor
		flags      byte
		encode     func([]byte) []byte
	}{
		{name: "stream", encode: func(b []byte) []byte { return b }},
		{name: "compressed", compressor: SnappyCompressor{MaxDecodedLen: 1 << 10}, flags: flagCompress, encode: func(b []byte) []byte { return snappy.Encode(nil, b) }},
	}
	for _, dec := range decoders {
		t.Run(dec.name, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					frame := newFrame(v4Request, dec.flags, opQuery, dec.encode(tc.body))
					data, err := decodeFrame(t, frame, dec.compressor)
					if tc.wantErr != "" {
						assert.ErrorContains(t, err, tc.wantErr, "negative length should be rejected")
						return
					}
					assert.NoError(t, err, "valid frame should parse")
					assert.Equal(t, tc.want, data, "unexpected decoded frame")
				})
			}
		})
	}
}

func TestReadFramePkeyCount(t *testing.T) {
	preparedResult := func(pkeyCount int32, pkeyIndexes ...uint16) []byte {
		body := appendInt32(nil, resultKindPrepared)
		body = binary.BigEndian.AppendUint16(body, 16)
		body = append(body, make([]byte, 16)...) // prepared id
		body = appendInt32(body, 0)              // flags
		body = appendInt32(body, 0)              // column count
		body = appendInt32(body, pkeyCount)
		for _, i := range pkeyIndexes {
			body = binary.BigEndian.AppendUint16(body, i)
		}
		body = appendInt32(body, 0) // result metadata flags
		return appendInt32(body, 0) // result metadata column count
	}

	tests := []struct {
		name    string
		body    []byte
		want    []int
		wantErr string
	}{
		{
			name: "valid",
			body: preparedResult(2, 1, 0),
			want: []int{1, 0},
		},
		{
			name:    "negative",
			body:    preparedResult(-1),
			wantErr: "invalid pkey count: -1",
		},
		{
			name: "at limit",
			body: preparedResult(maxPkeyCount, make([]uint16, maxPkeyCount)...),
			want: make([]int, maxPkeyCount),
		},
		{
			name:    "above limit",
			body:    preparedResult(maxPkeyCount + 1),
			wantErr: "invalid pkey count: 65537",
		},
		{
			name:    "maximum int",
			body:    preparedResult(0x7fffffff),
			wantErr: "invalid pkey count: 2147483647",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := decodeFrame(t, newFrame(v4Response, 0, opResult, tc.body), nil)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "invalid pkey count should be rejected before allocating")
				return
			}
			assert.NoError(t, err, "valid frame should parse")
			result, _ := data["result"].(map[string]any)
			prepared, _ := result["prepared"].(map[string]any)
			reqMeta, _ := prepared["req_meta"].(map[string]any)
			assert.Equal(t, tc.want, reqMeta["pkey_columns"], "unexpected pkey columns")
		})
	}
}

func TestReadFrameSnappyDecodedLength(t *testing.T) {
	voidResult := appendInt32(nil, resultKindVoid)
	compressor := SnappyCompressor{MaxDecodedLen: len(voidResult)}

	tests := []struct {
		name    string
		body    []byte
		want    map[string]any
		wantErr error
	}{
		{
			name: "decoded length at limit",
			body: snappy.Encode(nil, voidResult),
			want: map[string]any{"result": map[string]any{"type": "void"}},
		},
		{
			name:    "decoded length above limit",
			body:    binary.AppendUvarint(nil, uint64(len(voidResult)+1)),
			wantErr: ErrFrameTooBig,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame := newFrame(v4Response, flagCompress, opResult, tc.body)
			data, err := decodeFrame(t, frame, compressor)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "oversized decoded length should be rejected before allocating")
				return
			}
			assert.NoError(t, err, "valid frame should parse")
			assert.Equal(t, tc.want, data, "unexpected decoded frame")
		})
	}
}
