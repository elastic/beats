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
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/mapstr"
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

	// maximum number of bytes to use when reassembling partial CRI/docker log lines.
	// A value of 0 means no limit is applied during reassembly.
	maxBytes int

	parseLine func(message *reader.Message, msg *logLine) error

	stripNewLine func(msg *reader.Message)

	logger *logp.Logger

	// state is the progress of a Next call interrupted by reader.ErrWouldBlock.
	state readState
}

// readState is the progress of reading a log line that spans several chunks
// (partial CRI/Docker lines), kept across calls interrupted by
// reader.ErrWouldBlock.
type readState struct {
	// bytes consumed so far from the wrapped reader, including skipped lines.
	bytes int
	// message holds the chunks joined so far, valid when joining is true.
	message reader.Message
	line    logLine
	// truncated is set when the max_bytes limit was reached while joining.
	truncated bool
	// joining is set when message is a partial line waiting for more chunks.
	joining bool
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
func New(r reader.Reader, stream string, partial bool, format string, CRIFlags bool, maxBytes int, logger *logp.Logger) *DockerJSONReader {
	reader := DockerJSONReader{
		stream:   stream,
		partial:  partial,
		reader:   r,
		criflags: CRIFlags,
		maxBytes: maxBytes,
		logger:   logger.Named("reader_docker_json"),
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

func NewContainerParser(r reader.Reader, config *ContainerJSONConfig, maxBytes int, logger *logp.Logger) *DockerJSONReader {
	reader := DockerJSONReader{
		stream:   config.Stream.String(),
		partial:  true,
		reader:   r,
		criflags: true,
		maxBytes: maxBytes,
		logger:   logger.Named("parser_container"),
	}

	switch config.Format {
	case Docker, JSONFile:
		reader.parseLine = reader.parseDockerJSONLog
	case CRI:
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
		return fmt.Errorf("parsing CRI timestamp: %w", err)
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
		tags := bytes.SplitSeq(log[i], []byte{':'})
		for tag := range tags {
			if len(tag) == 1 && tag[0] == 'P' {
				partial = true
			}
		}
		i++
	}

	msg.Partial = partial
	message.AddFields(mapstr.M{
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
		return fmt.Errorf("decoding docker JSON: %w", err)
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339, msg.Time)
	if err != nil {
		return fmt.Errorf("parsing docker timestamp: %w", err)
	}
	message.Ts = ts

	message.AddFields(mapstr.M{
		"stream": msg.Stream,
	})

	if len(msg.Attrs) > 0 {
		message.AddFields(mapstr.M{
			"docker": mapstr.M{
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
//
// If the wrapped reader returns reader.ErrWouldBlock while a partial line is
// being joined (or after lines that were skipped), the progress made so far is
// kept in the reader and resumed by the next call, which returns
// reader.ErrWouldBlock with an empty message. This keeps the already consumed
// chunks from being returned as a standalone message, or lost, when the caller
// retains the reader across non-blocking reads.
func (p *DockerJSONReader) Next() (reader.Message, error) {
	st := p.state
	p.state = readState{}

	for {
		if !st.joining {
			message, err := p.reader.Next()

			// keep the right bytes count even if we return an error
			st.bytes += message.Bytes
			message.Bytes = st.bytes

			if err != nil {
				return p.fail(&st, message, err)
			}

			st.message = message
			st.line = logLine{}
			st.truncated = false
			err = p.parseLine(&st.message, &st.line)
			if err != nil {
				p.logger.Errorf("Parse line error: %v", err)
				continue
			}
		}

		// Handle multiline messages, join partial lines
		for p.partial && st.line.Partial {
			next, err := p.reader.Next()

			// keep the right bytes count even if we return an error
			st.bytes += next.Bytes
			st.message.Bytes = st.bytes

			if err != nil {
				st.joining = true
				return p.fail(&st, st.message, err)
			}
			err = p.parseLine(&next, &st.line)
			if err != nil {
				p.logger.Errorf("Parse line error: %v", err)
				continue
			}

			p.appendChunk(&st, next)
		}
		st.joining = false

		if p.stream != "all" && p.stream != st.line.Stream {
			continue
		}

		return st.message, nil
	}
}

// fail handles an error returned by the wrapped reader. reader.ErrWouldBlock
// parks the state for the next call and returns an empty message; any other
// error is returned with the message holding the bytes consumed so far.
func (p *DockerJSONReader) fail(st *readState, message reader.Message, err error) (reader.Message, error) {
	if errors.Is(err, reader.ErrWouldBlock) {
		p.state = *st
		return reader.Message{}, err
	}
	return message, err
}

// appendChunk appends the parsed chunk next to the partial line in st.
func (p *DockerJSONReader) appendChunk(st *readState, next reader.Message) {
	message := &st.message

	// Enforce max_bytes during partial line reassembly to prevent unbounded
	// memory growth. Once the limit is reached, drain remaining partial
	// chunks (updating the byte counter only) so the reader stays aligned
	// to logical line boundaries for the next Next() call.
	if st.truncated {
		return
	}
	if p.maxBytes > 0 && len(message.Content)+len(next.Content) > p.maxBytes {
		remaining := p.maxBytes - len(message.Content)
		if remaining < 0 {
			// Just cutting the slice won't shrink the underlying array,
			// so we create a slice the size we want and copy the data.
			msg := make([]byte, p.maxBytes)
			copy(msg, message.Content[:p.maxBytes])
			message.Content = msg
		} else if remaining > 0 {
			message.Content = append(message.Content, next.Content[:remaining]...)
			// Make sure the underlaying array is the same size as the content
			msg := make([]byte, p.maxBytes)
			copy(msg, message.Content)
			message.Content = msg
		}
		_ = message.AddFlagsWithKey("log.flags", "truncated")
		st.truncated = true
		return
	}
	message.Content = append(message.Content, next.Content...)
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

func (p *DockerJSONReader) Close() error {
	return p.reader.Close()
}

// SetReadDeadline delegates to the wrapped reader (see reader.DeadlineSetter).
func (p *DockerJSONReader) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.reader, t)
}
