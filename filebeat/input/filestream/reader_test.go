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

package filestream

import (
	"compress/gzip"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/reader"
)

type readerResponse struct {
	msg     string
	private any
	err     error
}
type mockReader struct {
	resp []readerResponse
	pos  int
}

func (r *mockReader) Next() (reader.Message, error) {
	if r.pos >= len(r.resp) {
		return reader.Message{}, io.EOF
	}
	resp := r.resp[r.pos]
	r.pos++

	return reader.Message{
		Content: []byte(resp.msg), Bytes: len(resp.msg), Private: resp.private}, resp.err
}

func (r *mockReader) Close() error {
	return nil
}

func TestLookaheadReader(t *testing.T) {
	testCases := map[string]struct {
		responses   []readerResponse
		wantResults []readerResponse
		eofErr      error
	}{
		"empty_reader": {
			wantResults: []readerResponse{{err: io.EOF}},
		},
		"eof_on_constructor": {
			responses:   []readerResponse{{err: io.EOF}},
			wantResults: []readerResponse{{err: io.EOF}},
		},
		"custom_eof_on_constructor": {
			responses: []readerResponse{{msg: "1st msg", err: gzip.ErrChecksum}},
			wantResults: []readerResponse{
				{msg: "1st msg", private: io.EOF, err: gzip.ErrChecksum},
				{err: io.EOF}},
			eofErr: gzip.ErrChecksum,
		},
		"single_message": {
			responses: []readerResponse{
				{msg: "single msg"},
				{err: io.EOF},
			},
			wantResults: []readerResponse{
				{msg: "single msg", private: io.EOF},
				{err: io.EOF},
			},
		},
		"multiple_messages": {
			responses: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg"},
				{err: io.EOF},
			},
			wantResults: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg", private: io.EOF},
				{err: io.EOF},
			},
		},
		"error_after_messages": {
			responses: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg", err: errors.New("partial read")},
				{msg: "3rd msg"},
				{err: io.EOF},
			},
			wantResults: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg", err: errors.New("partial read")},
				{msg: "3rd msg", private: io.EOF},
				{err: io.EOF},
			},
		},
		"overwrite_private_field": {
			responses: []readerResponse{
				{msg: "single msg", private: "some private value"},
				{err: io.EOF},
			},
			wantResults: []readerResponse{
				{msg: "single msg", private: io.EOF},
				{err: io.EOF},
			},
		},
		"consider_gzip.ErrChecksum_EOF": {
			responses: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg", err: errors.New("partial read")},
				{msg: "3rd msg", err: gzip.ErrChecksum},
				{err: io.EOF},
			},
			wantResults: []readerResponse{
				{msg: "1st msg"},
				{msg: "2nd msg", err: errors.New("partial read")},
				{msg: "3rd msg", private: io.EOF, err: gzip.ErrChecksum},
				{err: io.EOF},
			},
			eofErr: gzip.ErrChecksum,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mock := &mockReader{
				resp: tc.responses,
			}

			r := NewEOFLookaheadReader(mock, tc.eofErr)
			defer r.Close()

			var got []readerResponse
			for i := 0; i < len(tc.wantResults); i++ {
				gotMsg, gotErr := r.Next()
				got = append(got, readerResponse{
					msg:     string(gotMsg.Content),
					private: gotMsg.Private,
					err:     gotErr})
			}
			assert.Equal(t, tc.wantResults, got)
		})
	}
}
