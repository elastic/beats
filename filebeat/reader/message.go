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

package reader

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Message represents a reader event with timestamp, content and actual number
// of bytes read from input before decoding.
type Message struct {
	Ts      time.Time     // timestamp the content was read
	Content []byte        // actual content read
	Bytes   int           // total number of bytes read to generate the message
	Fields  common.MapStr // optional fields that can be added by reader
}

// IsEmpty returns true in case the message is empty
// A message with only newline character is counted as an empty message
func (m *Message) IsEmpty() bool {
	// If no Bytes were read, event is empty
	// For empty line Bytes is at least 1 because of the newline char
	if m.Bytes == 0 {
		return true
	}

	// Content length can be 0 because of JSON events. Content and Fields must be empty.
	if len(m.Content) == 0 && len(m.Fields) == 0 {
		return true
	}

	return false
}

func (msg *Message) AddFields(fields common.MapStr) {
	if fields == nil {
		return
	}

	if msg.Fields == nil {
		msg.Fields = common.MapStr{}
	}
	msg.Fields.Update(fields)
}
