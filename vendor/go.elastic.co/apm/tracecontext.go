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

package apm // import "go.elastic.co/apm"

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

const (
	elasticTracestateVendorKey = "es"
)

var (
	errZeroTraceID = errors.New("zero trace-id is invalid")
	errZeroSpanID  = errors.New("zero span-id is invalid")
)

// tracestateKeyRegexp holds a regular expression used for validating
// tracestate keys according to the standard rules:
//
//   key = lcalpha 0*255( lcalpha / DIGIT / "_" / "-"/ "*" / "/" )
//   key = ( lcalpha / DIGIT ) 0*240( lcalpha / DIGIT / "_" / "-"/ "*" / "/" ) "@" lcalpha 0*13( lcalpha / DIGIT / "_" / "-"/ "*" / "/" )
//   lcalpha = %x61-7A ; a-z
//
// nblkchr is used for defining valid runes for tracestate values.
var (
	tracestateKeyRegexp = regexp.MustCompile(`^[a-z](([a-z0-9_*/-]{0,255})|([a-z0-9_*/-]{0,240}@[a-z][a-z0-9_*/-]{0,13}))$`)

	nblkchr = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x21, 0x2B, 1},
			{0x2D, 0x3C, 1},
			{0x3E, 0x7E, 1},
		},
		LatinOffset: 3,
	}
)

const (
	traceOptionsRecordedFlag = 0x01
)

// TraceContext holds trace context for an incoming or outgoing request.
type TraceContext struct {
	// Trace identifies the trace forest.
	Trace TraceID

	// Span identifies a span: the parent span if this context
	// corresponds to an incoming request, or the current span
	// if this is an outgoing request.
	Span SpanID

	// Options holds the trace options propagated by the parent.
	Options TraceOptions

	// State holds the trace state.
	State TraceState
}

// TraceID identifies a trace forest.
type TraceID [16]byte

// Validate validates the trace ID.
// This will return non-nil for a zero trace ID.
func (id TraceID) Validate() error {
	if id.isZero() {
		return errZeroTraceID
	}
	return nil
}

func (id TraceID) isZero() bool {
	return id == (TraceID{})
}

// String returns id encoded as hex.
func (id TraceID) String() string {
	text, _ := id.MarshalText()
	return string(text)
}

// MarshalText returns id encoded as hex, satisfying encoding.TextMarshaler.
func (id TraceID) MarshalText() ([]byte, error) {
	text := make([]byte, hex.EncodedLen(len(id)))
	hex.Encode(text, id[:])
	return text, nil
}

// SpanID identifies a span within a trace.
type SpanID [8]byte

// Validate validates the span ID.
// This will return non-nil for a zero span ID.
func (id SpanID) Validate() error {
	if id.isZero() {
		return errZeroSpanID
	}
	return nil
}

func (id SpanID) isZero() bool {
	return id == SpanID{}
}

// String returns id encoded as hex.
func (id SpanID) String() string {
	text, _ := id.MarshalText()
	return string(text)
}

// MarshalText returns id encoded as hex, satisfying encoding.TextMarshaler.
func (id SpanID) MarshalText() ([]byte, error) {
	text := make([]byte, hex.EncodedLen(len(id)))
	hex.Encode(text, id[:])
	return text, nil
}

// TraceOptions describes the options for a trace.
type TraceOptions uint8

// Recorded reports whether or not the transaction/span may have been (or may be) recorded.
func (o TraceOptions) Recorded() bool {
	return (o & traceOptionsRecordedFlag) == traceOptionsRecordedFlag
}

// WithRecorded changes the "recorded" flag, and returns the new options
// without modifying the original value.
func (o TraceOptions) WithRecorded(recorded bool) TraceOptions {
	if recorded {
		return o | traceOptionsRecordedFlag
	}
	return o & (0xFF ^ traceOptionsRecordedFlag)
}

// TraceState holds vendor-specific state for a trace.
type TraceState struct {
	head *TraceStateEntry

	// Fields related to parsing the Elastic ("es") tracestate entry.
	//
	// These must not be modified after NewTraceState returns.
	parseElasticTracestateError error
	haveSampleRate              bool
	sampleRate                  float64
}

// NewTraceState returns a TraceState based on entries.
func NewTraceState(entries ...TraceStateEntry) TraceState {
	out := TraceState{}
	var last *TraceStateEntry
	for _, e := range entries {
		e := e // copy
		if last == nil {
			out.head = &e
		} else {
			last.next = &e
		}
		last = &e
	}
	for _, e := range entries {
		if e.Key != elasticTracestateVendorKey {
			continue
		}
		out.parseElasticTracestateError = out.parseElasticTracestate(e)
		break
	}
	return out
}

// parseElasticTracestate parses an Elastic ("es") tracestate entry.
//
// Per https://github.com/elastic/apm/blob/master/specs/agents/tracing-distributed-tracing.md,
// the "es" tracestate value format is: "key:value;key:value...". Unknown keys are ignored.
func (s *TraceState) parseElasticTracestate(e TraceStateEntry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	value := e.Value
	for value != "" {
		kv := value
		end := strings.IndexRune(value, ';')
		if end >= 0 {
			kv = value[:end]
			value = value[end+1:]
		} else {
			value = ""
		}
		sep := strings.IndexRune(kv, ':')
		if sep == -1 {
			return errors.New("malformed 'es' tracestate entry")
		}
		k, v := kv[:sep], kv[sep+1:]
		switch k {
		case "s":
			sampleRate, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			if sampleRate < 0 || sampleRate > 1 {
				return fmt.Errorf("sample rate %q out of range", v)
			}
			s.sampleRate = sampleRate
			s.haveSampleRate = true
		}
	}
	return nil
}

// String returns s as a comma-separated list of key-value pairs.
func (s TraceState) String() string {
	if s.head == nil {
		return ""
	}
	var buf bytes.Buffer
	s.head.writeBuf(&buf)
	for e := s.head.next; e != nil; e = e.next {
		buf.WriteByte(',')
		e.writeBuf(&buf)
	}
	return buf.String()
}

// Validate validates the trace state.
//
// This will return non-nil if any entries are invalid,
// if there are too many entries, or if an entry key is
// repeated.
func (s TraceState) Validate() error {
	if s.head == nil {
		return nil
	}
	recorded := make(map[string]int)
	var i int
	for e := s.head; e != nil; e = e.next {
		if i == 32 {
			return errors.New("tracestate contains more than the maximum allowed number of entries, 32")
		}
		if e.Key == elasticTracestateVendorKey {
			// s.parseElasticTracestateError holds a general e.Validate error if any
			// occurred, or any other error specific to the Elastic tracestate format.
			if err := s.parseElasticTracestateError; err != nil {
				return errors.Wrapf(err, "invalid tracestate entry at position %d", i)
			}
		} else {
			if err := e.Validate(); err != nil {
				return errors.Wrapf(err, "invalid tracestate entry at position %d", i)
			}
		}
		if prev, ok := recorded[e.Key]; ok {
			return fmt.Errorf("duplicate tracestate key %q at positions %d and %d", e.Key, prev, i)
		}
		recorded[e.Key] = i
		i++
	}
	return nil
}

// TraceStateEntry holds a trace state entry: a key/value pair
// representing state for a vendor.
type TraceStateEntry struct {
	next *TraceStateEntry

	// Key holds a vendor (and optionally, tenant) ID.
	Key string

	// Value holds a string representing trace state.
	Value string
}

func (e *TraceStateEntry) writeBuf(buf *bytes.Buffer) {
	buf.WriteString(e.Key)
	buf.WriteByte('=')
	buf.WriteString(e.Value)
}

// Validate validates the trace state entry.
//
// This will return non-nil if either the key or value is invalid.
func (e *TraceStateEntry) Validate() error {
	if !tracestateKeyRegexp.MatchString(e.Key) {
		return fmt.Errorf("invalid key %q", e.Key)
	}
	if err := e.validateValue(); err != nil {
		return errors.Wrapf(err, "invalid value for key %q", e.Key)
	}
	return nil
}

func (e *TraceStateEntry) validateValue() error {
	if e.Value == "" {
		return errors.New("value is empty")
	}
	runes := []rune(e.Value)
	n := len(runes)
	if n > 256 {
		return errors.Errorf("value contains %d characters, maximum allowed is 256", n)
	}
	if !unicode.In(runes[n-1], nblkchr) {
		return errors.Errorf("value contains invalid character %q", runes[n-1])
	}
	for _, r := range runes[:n-1] {
		if r != 0x20 && !unicode.In(r, nblkchr) {
			return errors.Errorf("value contains invalid character %q", r)
		}
	}
	return nil
}

func formatElasticTracestateValue(sampleRate float64) string {
	// 0       -> "s:0"
	// 1       -> "s:1"
	// 0.55555 -> "s:0.5555" (any rounding should be applied prior)
	return fmt.Sprintf("s:%.4g", sampleRate)
}
