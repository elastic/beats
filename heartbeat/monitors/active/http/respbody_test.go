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

// capturingValidator returns a bodyValidator that stores the body it receives, and a pointer
// to that stored value. Use it to assert how many bytes processBody passed to a validator.
func capturingValidator() (bodyValidator, *string) {
	var captured string
	return func(_ *http.Response, body string) error {
		captured = body
		return nil
	}, &captured
}

func fakeResponse(body string) *http.Response {
	return &http.Response{Body: io.NopCloser(strings.NewReader(body))}
}

// TestBodyMaxBytesCapsBodiesSentToValidators verifies that body validators receive at most
// CheckBodyMaxBytes bytes, regardless of the actual response size.
func TestBodyMaxBytesCapsBodiesSentToValidators(t *testing.T) {
	const limit = 200
	v, captured := capturingValidator()
	validator := multiValidator{bodyValidators: []bodyValidator{v}}

	processBody(
		fakeResponse(strings.Repeat("x", 3*limit)),
		responseConfig{IncludeBody: "never", CheckBodyMaxBytes: limit, IncludeBodyMaxBytes: 2048},
		validator,
	) //nolint: errcheck //testcode

	assert.Len(t, *captured, limit, "validator must receive exactly BodyMaxBytes bytes")
}

// TestBodyMaxBytesDoesNotTruncateSmallBodies verifies that a response body smaller than
// CheckBodyMaxBytes is passed to validators in full.
func TestBodyMaxBytesDoesNotTruncateSmallBodies(t *testing.T) {
	const limit = 1024
	body := strings.Repeat("x", limit/2)
	v, captured := capturingValidator()
	validator := multiValidator{bodyValidators: []bodyValidator{v}}

	processBody(
		fakeResponse(body),
		responseConfig{IncludeBody: "never", CheckBodyMaxBytes: limit, IncludeBodyMaxBytes: 2048},
		validator,
	) //nolint: errcheck //testcode

	assert.Equal(t, body, *captured, "validator must receive the full body when it fits within BodyMaxBytes")
}

// TestBodyMaxBytesMinimalBufferWithNoValidatorsAndNoIncludeBody verifies that when no body
// validators are registered and include_body is "never", processBody allocates only a minimal
// prefix buffer (minBufferBodyBytes). The bytes field must still reflect the full response size.
func TestBodyMaxBytesMinimalBufferWithNoValidatorsAndNoIncludeBody(t *testing.T) {
	const bodyMaxBytes = 1024
	body := strings.Repeat("x", 2*bodyMaxBytes)

	resp := fakeResponse(body)
	defer resp.Body.Close()

	fields, _, _ := processBody(
		resp,
		responseConfig{IncludeBody: "never", CheckBodyMaxBytes: bodyMaxBytes, IncludeBodyMaxBytes: 2048},
		multiValidator{},
	) //nolint: errcheck //testcode

	assert.Equal(t, len(body), fields["bytes"], "bytes must reflect the full response length")
	_, hasContent := fields["content"]
	assert.False(t, hasContent, "content must be absent when include_body is never")
}

func TestBodyMaxBytesIncludeBodyDefaultSizeLessThanBodyMax(t *testing.T) {
	const (
		bodyMaxBytes        = 1024
		includeBodyMaxBytes = 200 // default-like value, less than bodyMaxBytes
	)
	body := strings.Repeat("x", 2*bodyMaxBytes)

	resp := fakeResponse(body)
	defer resp.Body.Close()

	fields, _, _ := processBody(
		resp,
		responseConfig{IncludeBody: "always", CheckBodyMaxBytes: bodyMaxBytes, IncludeBodyMaxBytes: includeBodyMaxBytes},
		multiValidator{}, // no body validators — triggers the bug
	) //nolint: errcheck //testcode

	content, ok := fields["content"]
	require.True(t, ok, "content must be present when include_body is always")
	assert.Len(t, content, includeBodyMaxBytes,
		"event content should be IncludeBodyMaxBytes bytes; if it is %d (minBufferBodyBytes) the IncludeBody buffer condition is using the wrong comparator",
		minBufferBodyBytes)
}

// TestBodyMaxBytesExpandsWhenIncludeBodyMaxBytesIsLarger verifies that when
// IncludeBodyMaxBytes > BodyMaxBytes and include_body is active, the buffer expands so
// that the event content is not silently truncated below the configured limit.
func TestBodyMaxBytesExpandsWhenIncludeBodyMaxBytesIsLarger(t *testing.T) {
	const (
		bodyMaxBytes        = 100
		includeBodyMaxBytes = 300 // explicitly larger than bodyMaxBytes
	)
	body := strings.Repeat("x", 500)
	v, captured := capturingValidator()
	validator := multiValidator{bodyValidators: []bodyValidator{v}}

	resp := fakeResponse(body)
	defer resp.Body.Close()

	fields, _, _ := processBody(
		resp,
		responseConfig{IncludeBody: "always", CheckBodyMaxBytes: bodyMaxBytes, IncludeBodyMaxBytes: includeBodyMaxBytes},
		validator,
	) //nolint: errcheck //testcode

	assert.Len(t, *captured, includeBodyMaxBytes, "validator must see includeBodyMaxBytes when it exceeds bodyMaxBytes")
	content, ok := fields["content"]
	require.True(t, ok)
	assert.Len(t, content, includeBodyMaxBytes, "event content must be capped at IncludeBodyMaxBytes")
}

// TestBodyMaxBytesEventContentTrimmedToIncludeBodyMaxBytes verifies that when validators
// are present and CheckBodyMaxBytes > IncludeBodyMaxBytes, validators see CheckBodyMaxBytes bytes
// while the event content is trimmed to IncludeBodyMaxBytes.
func TestBodyMaxBytesEventContentTrimmedToIncludeBodyMaxBytes(t *testing.T) {
	const (
		bodyMaxBytes        = 500
		includeBodyMaxBytes = 100
	)
	body := strings.Repeat("x", 1000)
	v, captured := capturingValidator()
	validator := multiValidator{bodyValidators: []bodyValidator{v}}

	resp := fakeResponse(body)
	defer resp.Body.Close()

	fields, _, _ := processBody(
		resp,
		responseConfig{IncludeBody: "always", CheckBodyMaxBytes: bodyMaxBytes, IncludeBodyMaxBytes: includeBodyMaxBytes},
		validator,
	) //nolint: errcheck //testcode

	assert.Len(t, *captured, bodyMaxBytes, "validator must receive BodyMaxBytes bytes")
	content, ok := fields["content"]
	require.True(t, ok)
	assert.Len(t, content, includeBodyMaxBytes, "event content must be trimmed to IncludeBodyMaxBytes")
}
