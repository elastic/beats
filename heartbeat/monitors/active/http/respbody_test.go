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
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

func Test_handleRespBody(t *testing.T) {
	matchingBodyValidator := checkBody([]match.Matcher{match.MustCompile("hello")}, nil)
	failingBodyValidator := checkBody([]match.Matcher{match.MustCompile("goodbye")}, nil)

	matchingComboValidator := multiValidator{bodyValidators: []bodyValidator{matchingBodyValidator}}
	failingComboValidator := multiValidator{bodyValidators: []bodyValidator{failingBodyValidator}}

	type args struct {
		resp           *http.Response
		mimeType       string
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
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			true,
		},
		{
			"on_error with success",
			args{
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "on_error", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			false,
		},
		{
			"always with error",
			args{
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			true,
		},
		{
			"always with success",
			args{
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "always", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			true,
		},
		{
			"never with error",
			args{
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				failingComboValidator,
			},
			true,
			false,
		},
		{
			"never with success",
			args{
				simpleHTTPResponse(), //nolint:bodyclose // closed later
				"text/plain; charset=utf-8",
				responseConfig{IncludeBody: "never", IncludeBodyMaxBytes: 3},
				matchingComboValidator,
			},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, mimeType, err := processBody(tt.args.resp, tt.args.responseConfig, tt.args.validator)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleRespBody() error = %v, wantErr %v", err, tt.wantErr)
			}
			if mimeType != tt.args.mimeType {
				t.Errorf("invalid mime type - got: '%v' - want: '%v'", mimeType, tt.args.mimeType)
			}

			bodyMatch := map[string]any{
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
				resp:           simpleHTTPResponse(), //nolint:bodyclose // closed later
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
			rc := io.NopCloser(strings.NewReader(tt.body))
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

			assert.NoError(t, err)
		})
	}
}

func simpleHTTPResponse() *http.Response {
	return &http.Response{Body: io.NopCloser(strings.NewReader("hello"))}
}

// plainBody hides the io.WriterTo of the underlying reader, like a real
// http.Response.Body, so copying from it needs a buffer.
type plainBody struct{ io.Reader }

func (plainBody) Close() error { return nil }

func newPlainBody(s string) io.ReadCloser { return plainBody{strings.NewReader(s)} }

// readPrefixAndHash runs on every check of every http monitor and the body is
// usually tiny, so it must not allocate copy buffers per call.
func Test_readPrefixAndHashDoesNotAllocateCopyBuffers(t *testing.T) {
	const content = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	reader := strings.NewReader(content)
	body := plainBody{reader}

	allocs := testing.AllocsPerRun(200, func() {
		reader.Reset(content)
		if _, _, _, err := readPrefixAndHash(body, 128); err != nil {
			t.Error(err)
		}
	})
	// Measured: 9 allocations with the pool, 11 without it (one copy buffer for
	// each of the two copies), so 10 fails if either buffer is allocated per call.
	assert.LessOrEqual(t, allocs, float64(10),
		"readPrefixAndHash made %v allocations per call, expected the copy buffers to be pooled", allocs)
}

func Test_readPrefixAndHashConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Go(func() {
			body := strings.Repeat(string([]byte{byte('a' + i%26)}), 70000+i)
			expected := sha256.Sum256([]byte(body))
			for range 20 {
				size, prefix, hash, err := readPrefixAndHash(newPlainBody(body), 10)
				assert.NoError(t, err)
				assert.Equal(t, len(body), size)
				assert.Equal(t, body[:10], prefix)
				assert.Equal(t, hex.EncodeToString(expected[:]), hash)
			}
		})
	}
	wg.Wait()
}

func BenchmarkReadPrefixAndHash(b *testing.B) {
	body := strings.Repeat("a", 100)
	b.ReportAllocs()
	for b.Loop() {
		if _, _, _, err := readPrefixAndHash(newPlainBody(body), 128); err != nil {
			b.Fatal(err)
		}
	}
}
