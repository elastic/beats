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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Message represents a reader event with timestamp, content and actual number
// of bytes read from input before decoding.
type Message struct {
	Ts      time.Time // timestamp the content was read
	Content []byte    // actual content read
	Bytes   int       // total number of bytes read to generate the message
	Fields  mapstr.M  // optional fields that can be added by reader
	Meta    mapstr.M  // deprecated
	Private interface{}
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

// AddFields adds fields to the message.
func (m *Message) AddFields(fields mapstr.M) {
	if fields == nil {
		return
	}

	if m.Fields == nil {
		m.Fields = mapstr.M{}
	}
	m.Fields.Update(fields)
}

// AddFlagsWithKey adds flags to the message with an arbitrary key.
// If the field does not exist, it is created.
func (m *Message) AddFlagsWithKey(key string, flags ...string) error {
	if len(flags) == 0 {
		return nil
	}

	if m.Fields == nil {
		m.Fields = mapstr.M{}
	}

	return common.AddTagsWithKey(m.Fields, key, flags)
}

// ToEvent converts a Message to an Event that can be published
// to the output.
func (m *Message) ToEvent() beat.Event {

	if len(m.Content) > 0 {
		if m.Fields == nil {
			m.Fields = mapstr.M{}
		}
		m.Fields["message"] = string(m.Content)
	}

	return beat.Event{
		Timestamp: m.Ts,
		Meta:      m.Meta,
		Fields:    m.Fields,
		Private:   m.Private,
	}
}
