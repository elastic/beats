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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

func Test_handleRespBody(t *testing.T) {
	matchingBodyValidator := checkBody([]match.Matcher{match.MustCompile("hello")})
	failingBodyValidator := checkBody([]match.Matcher{match.MustCompile("goodbye")})

	matchingComboValidator := multiValidator{bodyValidators: []bodyValidator{matchingBodyValidator}}
	failingComboValidator := multiValidator{bodyValidators: []bodyValidator{failingBodyValidator}}

	type args struct {
		resp           *http.Response
		responseConfig responseConfig
		validator      multiValidator
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
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			true,
		},
		{
			"on_error with success",
			args{
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			false,
		},
		{
			"always with error",
			args{
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			true,
		},
		{
			"always with success",
			args{
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			true,
		},
		{
			"never with error",
			args{
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			false,
		},
		{
			"never with success",
			args{
				simpleHTTPResponse("hello"),
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := processBody(tt.args.resp, tt.args.responseConfig, tt.args.validator)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleRespBody() error = %v, wantErr %v", err, tt.wantErr)
			}

			bodyMatch := map[string]interface{}{
				"hash":  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
				"bytes": 5,
			}
			if tt.wantFieldsSet {
				bodyMatch["content"] = "hel"
			}

			testslike.Test(t, lookslike.MustCompile(bodyMatch), fields)
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
		wantBodySize   int
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBodySample, gotBodySize, gotHashStr, err := readBody(tt.args.resp, tt.args.maxSampleBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("readBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotBodySample != tt.wantBodySample {
				t.Errorf("readBody() gotBodySample = %v, want %v", gotBodySample, tt.wantBodySample)
			}
			if gotBodySize != tt.wantBodySize {
				t.Errorf("readBody() gotBodySize = %v, want %v", gotBodySize, tt.wantBodySize)
			}
			if gotHashStr != tt.wantHashStr {
				t.Errorf("readBody() gotHashStr = %v, want %v", gotHashStr, tt.wantHashStr)
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

			assert.Equal(t, len(tt.body), gotRespSize)
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
