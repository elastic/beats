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

package readjson

import (
	"bytes"
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/reader"
)

// DockerJSONReader processor renames a given field
type DockerJSONReader struct {
	reader reader.Reader
	// stream filter, `all`, `stderr` or `stdout`
	stream string

	// join partial lines
	partial bool

	// parse CRI flags
	criflags bool

	parseLine func(message *reader.Message, msg *logLine) error

	stripNewLine func(msg *reader.Message)
}

type logLine struct {
	Partial   bool              `json:"-"`
	Timestamp time.Time         `json:"-"`
	Time      string            `json:"time"`
	Stream    string            `json:"stream"`
	Log       string            `json:"log"`
	Attrs     map[string]string `json:"attrs"`
}

// New creates a new reader renaming a field
func New(r reader.Reader, stream string, partial bool, format string, CRIFlags bool) *DockerJSONReader {
	reader := DockerJSONReader{
		stream:   stream,
		partial:  partial,
		reader:   r,
		criflags: CRIFlags,
	}

	switch strings.ToLower(format) {
	case "docker", "json-file":
		reader.parseLine = reader.parseDockerJSONLog
	case "cri":
		reader.parseLine = reader.parseCRILog
	default:
		reader.parseLine = reader.parseAuto
	}

	if runtime.GOOS == "windows" {
		reader.stripNewLine = stripNewLineWin
	} else {
		reader.stripNewLine = stripNewLine
	}

	return &reader
}

// parseCRILog parses logs in CRI log format.
// CRI log format example :
// 2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
func (p *DockerJSONReader) parseCRILog(message *reader.Message, msg *logLine) error {
	split := 3
	// read line tags if split is enabled:
	if p.criflags {
		split = 4
	}

	// current field
	i := 0

	// timestamp
	log := bytes.SplitN(message.Content, []byte{' '}, split)
	if len(log) < split {
		return errors.New("invalid CRI log format")
	}
	ts, err := time.Parse(time.RFC3339Nano, string(log[i]))
	if err != nil {
		return errors.Wrap(err, "parsing CRI timestamp")
	}
	message.Ts = ts
	i++

	// stream
	msg.Stream = string(log[i])
	i++

	// tags
	partial := false
	if p.criflags {
		// currently only P(artial) or F(ull) are available
		tags := bytes.Split(log[i], []byte{':'})
		for _, tag := range tags {
			if len(tag) == 1 && tag[0] == 'P' {
				partial = true
			}
		}
		i++
	}

	msg.Partial = partial
	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	// Remove \n ending for partial messages
	message.Content = log[i]
	if partial {
		p.stripNewLine(message)
	}

	return nil
}

// parseReaderLog parses logs in Docker JSON log format.
// Docker JSON log format example:
// {"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}
func (p *DockerJSONReader) parseDockerJSONLog(message *reader.Message, msg *logLine) error {
	dec := json.NewDecoder(bytes.NewReader(message.Content))

	if err := dec.Decode(&msg); err != nil {
		return errors.Wrap(err, "decoding docker JSON")
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339, msg.Time)
	if err != nil {
		return errors.Wrap(err, "parsing docker timestamp")
	}
	message.Ts = ts

	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})

	if len(msg.Attrs) > 0 {
		message.AddFields(common.MapStr{
			"docker": common.MapStr{
				"attrs": msg.Attrs,
			},
		})
	}

	message.Content = []byte(msg.Log)
	msg.Partial = (len(message.Content) == 0) || (message.Content[len(message.Content)-1] != byte('\n'))
	return nil
}

func (p *DockerJSONReader) parseAuto(message *reader.Message, msg *logLine) error {
	if len(message.Content) > 0 && message.Content[0] == '{' {
		return p.parseDockerJSONLog(message, msg)
	}

	return p.parseCRILog(message, msg)
}

// Next returns the next line.
func (p *DockerJSONReader) Next() (reader.Message, error) {
	var bytes int
	for {
		message, err := p.reader.Next()

		// keep the right bytes count even if we return an error
		bytes += message.Bytes
		message.Bytes = bytes

		if err != nil {
			return message, err
		}

		var logLine logLine
		err = p.parseLine(&message, &logLine)
		if err != nil {
			logp.Err("Parse line error: %v", err)
			return message, reader.ErrLineUnparsable
		}

		// Handle multiline messages, join partial lines
		for p.partial && logLine.Partial {
			next, err := p.reader.Next()

			// keep the right bytes count even if we return an error
			bytes += next.Bytes
			message.Bytes = bytes

			if err != nil {
				return message, err
			}
			err = p.parseLine(&next, &logLine)
			if err != nil {
				logp.Err("Parse line error: %v", err)
				return message, reader.ErrLineUnparsable
			}
			message.Content = append(message.Content, next.Content...)
		}

		if p.stream != "all" && p.stream != logLine.Stream {
			continue
		}

		return message, err
	}
}

func stripNewLine(msg *reader.Message) {
	l := len(msg.Content)
	if l > 0 && msg.Content[l-1] == '\n' {
		msg.Content = msg.Content[:l-1]
	}
}

func stripNewLineWin(msg *reader.Message) {
	msg.Content = bytes.TrimRightFunc(msg.Content, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}
