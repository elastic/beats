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

// Package readertest provides test doubles for reader.Reader implementations.
package readertest

import (
	"io"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

// ScriptStep is one result of ScriptedReader.Next: either a line or an error.
type ScriptStep struct {
	Line string
	Err  error
}

// LineStep returns a ScriptStep that yields line as a Message.
func LineStep(line string) ScriptStep {
	return ScriptStep{Line: line}
}

// Error steps for ScriptedReader.
var (
	WouldBlockStep = ScriptStep{Err: reader.ErrWouldBlock}
	EOFStep        = ScriptStep{Err: io.EOF}
)

// ScriptedReader replays a sequence of lines and errors like a non-blocking
// source. Once the script is exhausted it returns io.EOF.
type ScriptedReader struct {
	steps []ScriptStep
}

var _ reader.Reader = (*ScriptedReader)(nil)

// NewScriptedReader creates a ScriptedReader initialized with steps.
func NewScriptedReader(steps ...ScriptStep) *ScriptedReader {
	return &ScriptedReader{steps: steps}
}

// Push appends steps to the end of the script.
func (s *ScriptedReader) Push(steps ...ScriptStep) {
	s.steps = append(s.steps, steps...)
}

// Next returns the next scripted message or error, or io.EOF when exhausted.
func (s *ScriptedReader) Next() (reader.Message, error) {
	if len(s.steps) == 0 {
		return reader.Message{}, io.EOF
	}
	step := s.steps[0]
	s.steps = s.steps[1:]
	if step.Err != nil {
		return reader.Message{}, step.Err
	}
	return reader.Message{
		Ts:      time.Now(),
		Content: []byte(step.Line),
		Bytes:   len(step.Line) + 1, // + newline
		Fields:  mapstr.M{},
	}, nil
}

// Close implements reader.Reader.
func (s *ScriptedReader) Close() error { return nil }
