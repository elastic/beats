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

package mime

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/h2non/filetype"
)

const (
	// size for mime detection, office file
	// detection requires ~8kb to detect properly
	maxHeaderSize = 8192
)

// DetectBytes tries to detect a mime-type based off
// of a chunk of bytes passed into the function
func DetectBytes(data []byte) string {
	header := data
	if len(data) > maxHeaderSize {
		header = data[:maxHeaderSize]
	}
	kind, err := filetype.Match(header)
	if err == nil && kind != filetype.Unknown {
		// we have a known filetype, return
		return kind.MIME.Value
	}
	// if the above fails, try and sniff with http sniffing
	netType := http.DetectContentType(header)
	// try and parse any sort of text as json or xml
	if strings.HasPrefix(netType, "text/plain") {
		if detected := detectEncodedText(data); detected != "" {
			return detected
		}
	}
	// The fallback for http.DetectContentType is "application/octet-stream"
	// meaning that if we see it, we were unable to determine the type and
	// we just know we're dealing with a chunk of some sort of bytes. Rather
	// than reporting the fallback, we'll just say we were unable to detect
	// the type.
	if netType == "application/octet-stream" {
		return ""
	}
	return netType
}

func detectEncodedText(data []byte) string {
	// figure out how to optimize this so we don't have to try and parse the whole payload
	// every time
	if json.Valid(data) {
		return "application/json"
	}
	if xml.Unmarshal(data, new(interface{})) == nil {
		return "text/xml"
	}
	return ""
}
