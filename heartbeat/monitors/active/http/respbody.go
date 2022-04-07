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

	"github.com/docker/go-units"

	"github.com/elastic/beats/v8/heartbeat/reason"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/mime"
)

const (
	// maxBufferBodyBytes sets a hard limit on how much we're willing to buffer for any reason internally.
	// since we must buffer the whole body for body validators this is effectively a cap on that.
	// 100MiB out to be enough for everybody.
	maxBufferBodyBytes = 100 * units.MiB
	// minBufferBytes is the lower bound on how much of the body content we actually read. In order to do
	// do mime type detection of the response body, we always need a small read buffer.
	minBufferBodyBytes = 128
)

func processBody(resp *http.Response, config responseConfig, validator multiValidator) (common.MapStr, string, reason.Reason) {
	// Determine how much of the body to actually buffer in memory
	var bufferBodyBytes int
	if validator.wantsBody() {
		bufferBodyBytes = maxBufferBodyBytes
	} else if config.IncludeBody == "always" || config.IncludeBody == "on_error" {
		// If the user has asked for bodies to be recorded we only need to buffer that much
		bufferBodyBytes = config.IncludeBodyMaxBytes
	}

	if bufferBodyBytes < minBufferBodyBytes {
		bufferBodyBytes = minBufferBodyBytes
	}

	respBody, bodyLenBytes, bodyHash, respErr := readBody(resp, bufferBodyBytes)
	// If we encounter an error while reading the body just fail early
	if respErr != nil {
		return nil, "", reason.IOFailed(fmt.Errorf("failed reading HTTP response body: %w", respErr))
	}

	// Run any validations
	errReason := validator.validate(resp, respBody)

	bodyFields := common.MapStr{
		"hash":  bodyHash,
		"bytes": bodyLenBytes,
	}
	if config.IncludeBody == "always" ||
		(config.IncludeBody == "on_error" && errReason != nil) {

		// Do not store more bytes than the config specifies. We may
		// have read extra bytes for the validators
		sampleNumBytes := len(respBody)
		if bodyLenBytes < sampleNumBytes {
			sampleNumBytes = bodyLenBytes
		}
		if config.IncludeBodyMaxBytes < sampleNumBytes {
			sampleNumBytes = config.IncludeBodyMaxBytes
		}

		bodyFields["content"] = respBody[0:sampleNumBytes]
	}

	return bodyFields, mime.Detect(respBody), errReason
}

// readBody reads the first sampleSize bytes from the httpResponse,
// then closes the body (which closes the connection). It doesn't return any errors
// but does log them. During an error case the return values will be (nil, -1).
// The maxBytes params controls how many bytes will be returned in a string, not how many will be read.
// We always read the full response here since we want to time downloading the full thing.
// This may return a nil body if the response is not valid UTF-8
func readBody(resp *http.Response, maxSampleBytes int) (bodySample string, bodySize int, hashStr string, err error) {
	defer resp.Body.Close()

	respSize, bodySample, hash, err := readPrefixAndHash(resp.Body, maxSampleBytes)

	return bodySample, respSize, hash, err
}

func readPrefixAndHash(body io.ReadCloser, maxPrefixSize int) (respSize int, prefix string, hashStr string, err error) {
	hash := sha256.New()

	var prefixBuf strings.Builder

	n, err := io.Copy(&prefixBuf, io.TeeReader(io.LimitReader(body, int64(maxPrefixSize)), hash))
	if err == nil {
		// finish streaming into hash if the body has not been fully consumed yet
		var m int64
		m, err = io.Copy(hash, body)
		n += m
	}

	// The ErrUnexpectedEOF message can be confusing to users, so we clarify it here
	if err == io.ErrUnexpectedEOF {
		return 0, "", "", fmt.Errorf("connection closed unexpectedly: %w", err)
	}

	// Note that io.EOF is distinct from io.ErrUnexpectedEOF
	if err != nil && err != io.EOF {
		return 0, "", "", err
	}

	return int(n), prefixBuf.String(), hex.EncodeToString(hash.Sum(nil)), nil
}
