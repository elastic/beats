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

package testing

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Logger is log interface that matches *testing.T.
type Logger interface {
	// Log logs the arguments.
	Log(args ...any)
	// Logf logs the formatted arguments.
	Logf(format string, args ...any)
}

// logWatcher is an `io.Writer` that processes the log lines outputted from the spawned Elastic Agent.
//
// `Write` handles parsing lines as either ndjson or plain text.
type logWatcher struct {
	remainder []byte
	replicate Logger
	alert     chan error
}

func newLogWatcher(replicate Logger) *logWatcher {
	return &logWatcher{
		replicate: replicate,
		alert:     make(chan error),
	}
}

// Watch returns the channel that will get an error when an error is identified from the log.
func (r *logWatcher) Watch() <-chan error {
	return r.alert
}

// Write implements the `io.Writer` interface.
func (r *logWatcher) Write(p []byte) (int, error) {
	if len(p) == 0 {
		// nothing to do
		return 0, nil
	}
	offset := 0
	for {
		idx := bytes.IndexByte(p[offset:], '\n')
		if idx < 0 {
			// not all used add to remainder to be used on next call
			r.remainder = append(r.remainder, p[offset:]...)
			return len(p), nil
		}

		var line []byte
		if r.remainder != nil {
			line = r.remainder
			r.remainder = nil
			line = append(line, p[offset:offset+idx]...)
		} else {
			line = append(line, p[offset:offset+idx]...)
		}
		offset += idx + 1
		// drop '\r' from line (needed for Windows)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[0 : len(line)-1]
		}
		if len(line) == 0 {
			// empty line
			continue
		}
		str := strings.TrimSpace(string(line))
		// try to parse line as JSON
		if str[0] == '{' && r.handleJSON(str) {
			// handled as JSON
			continue
		}
		// considered standard text being it's not JSON, just replicate
		if r.replicate != nil {
			r.replicate.Log(str)
		}
	}
}

func (r *logWatcher) handleJSON(line string) bool {
	var evt map[string]interface{}
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return false
	}
	if r.replicate != nil {
		r.replicate.Log(line)
	}
	lvl := getLevel(evt, "log.level")
	msg := getMessage(evt, "message")
	if lvl == logp.ErrorLevel {
		r.alert <- errors.New(msg)
	}
	return true
}

func getLevel(evt map[string]interface{}, key string) logp.Level {
	lvl := logp.InfoLevel
	err := unmarshalLevel(&lvl, getStrVal(evt, key))
	if err == nil {
		delete(evt, key)
	}
	return lvl
}

func unmarshalLevel(lvl *logp.Level, val string) error {
	if val == "" {
		return errors.New("empty val")
	} else if val == "trace" {
		// logp doesn't handle trace level we cast to debug
		*lvl = logp.DebugLevel
		return nil
	}
	return lvl.Unpack(val)
}

func getMessage(evt map[string]interface{}, key string) string {
	msg := getStrVal(evt, key)
	if msg != "" {
		delete(evt, key)
	}
	return msg
}

func getStrVal(evt map[string]interface{}, key string) string {
	raw, ok := evt[key]
	if !ok {
		return ""
	}
	str, ok := raw.(string)
	if !ok {
		return ""
	}
	return str
}
