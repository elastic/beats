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

package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/reason"
	"github.com/elastic/beats/libbeat/beat"
)

func Test_handleRespBody(t *testing.T) {
	type args struct {
		event          *beat.Event
		resp           *http.Response
		responseConfig responseConfig
		errReason      reason.Reason
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		wantFieldsSet bool
	}{
		{
			"on_error with error",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				reason.IOFailed(fmt.Errorf("something happened")),
			},
			false,
			true,
		},
		{
			"on_error with success",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				nil,
			},
			false,
			false,
		},
		{
			"always with error",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				reason.IOFailed(fmt.Errorf("something happened")),
			},
			false,
			true,
		},
		{
			"always with success",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				nil,
			},
			false,
			true,
		},
		{
			"never with error",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				reason.IOFailed(fmt.Errorf("something happened")),
			},
			false,
			false,
		},
		{
			"never with success",
			args{
				&beat.Event{},
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				nil,
			},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.args.event
			if err := handleRespBody(tt.args.event, tt.args.resp, tt.args.responseConfig, tt.args.errReason); (err != nil) != tt.wantErr {
				t.Errorf("handleRespBody() error = %v, wantErr %v", err, tt.wantErr)
			}

			bodyMatch := map[string]interface{}{
				"hash":  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
				"bytes": int64(5),
			}
			if tt.wantFieldsSet {
				bodyMatch["content"] = "hel"
			}

			testslike.Test(t,
				lookslike.MustCompile(
					map[string]interface{}{
						"http.response.body": bodyMatch,
					}),
				event.Fields)
		})
	}
}

func Test_readResp(t *testing.T) {
	type args struct {
		resp           *http.Response
		maxSampleBytes int
	}
	tests := []struct {
		name           string
		args           args
		wantBodySample string
		wantBodySize   int64
		wantHashStr    string
		wantErr        bool
	}{
		{
			name: "response exists",
			args: args{
				resp:           simpleHTTPResponse("hello"),
				maxSampleBytes: 3,
			},
			wantBodySample: "hel",
			wantBodySize:   5,
			wantHashStr:    "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr:        false,
		},
		{
			name: "no resp",
			args: args{
				resp:           nil,
				maxSampleBytes: 3,
			},
			wantBodySample: "",
			wantBodySize:   -1,
			wantHashStr:    "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBodySample, gotBodySize, gotHashStr, err := readResp(tt.args.resp, tt.args.maxSampleBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("readResp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotBodySample != tt.wantBodySample {
				t.Errorf("readResp() gotBodySample = %v, want %v", gotBodySample, tt.wantBodySample)
			}
			if gotBodySize != tt.wantBodySize {
				t.Errorf("readResp() gotBodySize = %v, want %v", gotBodySize, tt.wantBodySize)
			}
			if gotHashStr != tt.wantHashStr {
				t.Errorf("readResp() gotHashStr = %v, want %v", gotHashStr, tt.wantHashStr)
			}
		})
	}
}

func Test_readPrefixAndHash(t *testing.T) {
	type args struct {
		body          io.ReadCloser
		maxPrefixSize int
	}

	longBytes := make([]byte, 2*1024*1024) //2MiB
	for idx := range longBytes {
		longBytes[idx] = 'x'
	}
	longStr := string(longBytes)

	bodies := []struct {
		name string
		body string
	}{
		{"short", "short"},
		{"long", longStr},
		{"mb chars", "Hello, 世界"},
	}

	type testSpec struct {
		name string
		body string
		len  int
		err  bool
	}

	var tests []testSpec

	for _, bSpec := range bodies {
		add := func(name string, len int, err bool) {
			tests = append(tests,
				testSpec{
					fmt.Sprintf("%s/%s", bSpec.name, name),
					bSpec.body,
					len,
					err,
				},
			)
		}
		add("1 byte prefix", 1, false)
		add("multi read byte prefix", 1026, false)
		add("all byte prefix", len(bSpec.body), false)
		add("extra byte prefix", len(bSpec.body), false)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := ioutil.NopCloser(strings.NewReader(tt.body))
			gotRespSize, gotPrefix, gotHashStr, err := readPrefixAndHash(rc, tt.len)

			if tt.err {
				require.Error(t, err)
			}

			assert.Equal(t, int64(len(tt.body)), gotRespSize)
			if tt.len <= len(tt.body) {
				assert.Equal(t, tt.body[0:tt.len], gotPrefix)
			} else {
				assert.Equal(t, tt.body[0:len(tt.body)], gotPrefix)
			}

			expectedHash := sha256.Sum256([]byte(tt.body))
			assert.Equal(t, hex.EncodeToString(expectedHash[:]), gotHashStr)

			assert.Nil(t, err)
		})
	}
}

func simpleHTTPResponse(body string) *http.Response {
	return &http.Response{Body: ioutil.NopCloser(strings.NewReader(body))}
}
