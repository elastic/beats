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

	// parse CRI flags
	criflags bool
}

type logLine struct {
	Partial   bool      `json:"-"`
	Timestamp time.Time `json:"-"`
	Time      string    `json:"time"`
	Stream    string    `json:"stream"`
	Log       string    `json:"log"`
}

// New creates a new reader renaming a field
func New(r reader.Reader, stream string, partial bool, CRIFlags bool) *Reader {
	return &Reader{
		stream:   stream,
		partial:  partial,
		reader:   r,
		criflags: CRIFlags,
	}
}

// parseCRILog parses logs in CRI log format.
// CRI log format example :
// 2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
func (p *Reader) parseCRILog(message *reader.Message, msg *logLine) error {
	split := 3
	// read line tags if split is enabled:
	if p.criflags {
		split = 4
	}

	// current field
	i := 0

	// timestamp
	log := strings.SplitN(string(message.Content), " ", split)
	if len(log) < split {
		return errors.New("invalid CRI log format")
	}
	ts, err := time.Parse(time.RFC3339, log[i])
	if err != nil {
		return errors.Wrap(err, "parsing CRI timestamp")
	}
	message.Ts = ts
	i++

	// stream
	msg.Stream = log[i]
	i++

	// tags
	partial := false
	if p.criflags {
		// currently only P(artial) or F(ull) are available
		tags := strings.Split(log[i], ":")
		for _, tag := range tags {
			if tag == "P" {
				partial = true
			}
		}
		i++
	}

	msg.Partial = partial
	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	// Remove ending \n for partial messages
	message.Content = []byte(log[i])
	if partial {
		message.Content = bytes.TrimRightFunc(message.Content, func(r rune) bool {
			return r == '\n' || r == '\r'
		})
	}

	return nil
}

// parseReaderLog parses logs in Docker JSON log format.
// Docker JSON log format example:
// {"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}
func (p *Reader) parseDockerJSONLog(message *reader.Message, msg *logLine) error {
	dec := json.NewDecoder(bytes.NewReader(message.Content))

	if err := dec.Decode(&msg); err != nil {
		return errors.Wrap(err, "decoding docker JSON")
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339, msg.Time)
	if err != nil {
		return errors.Wrap(err, "parsing docker timestamp")
	}

	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	message.Content = []byte(msg.Log)
	message.Ts = ts
	msg.Partial = message.Content[len(message.Content)-1] != byte('\n')

	return nil
}

func (p *Reader) parseLine(message *reader.Message, msg *logLine) error {
	if strings.HasPrefix(string(message.Content), "{") {
		return p.parseDockerJSONLog(message, msg)
	}

	return p.parseCRILog(message, msg)
}

// Next returns the next line.
func (p *Reader) Next() (reader.Message, error) {
	for {
		message, err := p.reader.Next()
		if err != nil {
			return message, err
		}

		var logLine logLine
		err = p.parseLine(&message, &logLine)
		if err != nil {
			return message, err
		}

		// Handle multiline messages, join partial lines
		for p.partial && logLine.Partial {
			next, err := p.reader.Next()
			if err != nil {
				return message, err
			}
			err = p.parseLine(&next, &logLine)
			if err != nil {
				return message, err
			}
			message.Content = append(message.Content, next.Content...)
			message.Bytes += next.Bytes
		}

		if p.stream != "all" && p.stream != logLine.Stream {
			continue
		}

		return message, err
	}
}
