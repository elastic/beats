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
	"unicode/utf8"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/reason"
	"github.com/elastic/beats/libbeat/beat"
)

func handleRespBody(event *beat.Event, resp *http.Response, responseConfig responseConfig, errReason reason.Reason) error {
	defer resp.Body.Close()

	sampleMaxBytes := responseConfig.IncludeBodyMaxBytes

	includeSample := responseConfig.IncludeBody == "always" || (responseConfig.IncludeBody == "on_error" && errReason != nil)

	// No need to return any actual body bytes if we'll discard them anyway. This should save on allocation
	if !includeSample {
		sampleMaxBytes = 0
	}

	sampleStr, bodyBytes, bodyHash, err := readResp(resp, sampleMaxBytes)
	if err != nil {
		return err
	}

	evtBodyMap := common.MapStr{
		"hash":  bodyHash,
		"bytes": bodyBytes,
	}
	if includeSample {
		evtBodyMap["content"] = sampleStr
	}

	eventext.MergeEventFields(event, common.MapStr{
		"http": common.MapStr{
			"response": common.MapStr{"body": evtBodyMap},
		},
	})

	return nil
}

// readResp reads the first sampleSize bytes from the httpResponse,
// then closes the body (which closes the connection). It doesn't return any errors
// but does log them. During an error case the return values will be (nil, -1).
// The maxBytes params controls how many bytes will be returned in a string, not how many will be read.
// We always read the full response here since we want to time downloading the full thing.
// This may return a nil body if the response is not valid UTF-8
func readResp(resp *http.Response, maxSampleBytes int) (bodySample string, bodySize int64, hashStr string, err error) {
	if resp == nil {
		return "", -1, "", fmt.Errorf("cannot readResp of nil HTTP response")
	}

	respSize, bodySample, hash, err := readPrefixAndHash(resp.Body, maxSampleBytes)

	return bodySample, respSize, hash, err
}

func readPrefixAndHash(body io.ReadCloser, maxPrefixSize int) (respSize int64, prefix string, hashStr string, err error) {
	hash := sha256.New()
	// Function to lazily get the body of the response
	rawBuf := make([]byte, 1024)

	// Buffer to hold the prefix output along with tracking info
	prefixBuf := make([]byte, maxPrefixSize)
	prefixRemainingBytes := maxPrefixSize
	prefixWriteOffset := 0
	for {
		readSize, readErr := body.Read(rawBuf)

		respSize += int64(readSize)
		hash.Write(rawBuf[:readSize])

		if prefixRemainingBytes > 0 {
			if readSize >= prefixRemainingBytes {
				copy(prefixBuf[prefixWriteOffset:maxPrefixSize], rawBuf[:prefixRemainingBytes])
				prefixWriteOffset += prefixRemainingBytes
				prefixRemainingBytes = 0
			} else {
				copy(prefixBuf[prefixWriteOffset:prefixWriteOffset+readSize], rawBuf[:readSize])
				prefixWriteOffset += readSize
				prefixRemainingBytes -= readSize
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return 0, "", "", readErr
		}
	}

	// We discard the body if it is not valid UTF-8
	if utf8.Valid(prefixBuf[:prefixWriteOffset]) {
		prefix = string(prefixBuf[:prefixWriteOffset])
	}
	return respSize, prefix, hex.EncodeToString(hash.Sum(nil)), nil
}
