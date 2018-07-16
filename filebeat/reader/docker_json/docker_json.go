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

package docker_json

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/elastic/beats/filebeat/reader"
	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
)

// Reader processor renames a given field
type Reader struct {
	reader reader.Reader
	// stream filter, `all`, `stderr` or `stdout`
	stream string

	// join partial lines
	partial bool
}

type dockerLog struct {
	Timestamp string `json:"time"`
	Log       string `json:"log"`
	Stream    string `json:"stream"`
}

type crioLog struct {
	Timestamp time.Time
	Stream    string
	Log       []byte
}

// New creates a new reader renaming a field
func New(r reader.Reader, stream string, partial bool) *Reader {
	return &Reader{
		stream:  stream,
		partial: partial,
		reader:  r,
	}
}

// parseCRILog parses logs in CRI log format.
// CRI log format example :
// 2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
func parseCRILog(message reader.Message, msg *crioLog) (reader.Message, error) {
	log := strings.SplitN(string(message.Content), " ", 3)
	if len(log) < 3 {
		return message, errors.New("invalid CRI log")
	}
	ts, err := time.Parse(time.RFC3339, log[0])
	if err != nil {
		return message, errors.Wrap(err, "parsing CRI timestamp")
	}

	msg.Timestamp = ts
	msg.Stream = log[1]
	msg.Log = []byte(log[2])
	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	message.Content = msg.Log
	message.Ts = ts

	return message, nil
}

// parseReaderLog parses logs in Docker JSON log format.
// Docker JSON log format example:
// {"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}
func parseDockerJSONLog(message reader.Message, msg *dockerLog) (reader.Message, error) {
	dec := json.NewDecoder(bytes.NewReader(message.Content))
	if err := dec.Decode(&msg); err != nil {
		return message, errors.Wrap(err, "decoding docker JSON")
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err != nil {
		return message, errors.Wrap(err, "parsing docker timestamp")
	}

	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	message.Content = []byte(msg.Log)
	message.Ts = ts

	return message, nil
}

// Next returns the next line.
func (p *Reader) Next() (reader.Message, error) {
	for {
		message, err := p.reader.Next()
		if err != nil {
			return message, err
		}

		var dockerLine dockerLog
		var crioLine crioLog

		if strings.HasPrefix(string(message.Content), "{") {
			message, err = parseDockerJSONLog(message, &dockerLine)
			if err != nil {
				return message, err
			}
			// Handle multiline messages, join lines that don't end with \n
			for p.partial && message.Content[len(message.Content)-1] != byte('\n') {
				next, err := p.reader.Next()
				if err != nil {
					return message, err
				}
				next, err = parseDockerJSONLog(next, &dockerLine)
				if err != nil {
					return message, err
				}
				message.Content = append(message.Content, next.Content...)
			}
		} else {
			message, err = parseCRILog(message, &crioLine)
		}

		if p.stream != "all" && p.stream != dockerLine.Stream && p.stream != crioLine.Stream {
			continue
		}

		return message, err
	}
}
