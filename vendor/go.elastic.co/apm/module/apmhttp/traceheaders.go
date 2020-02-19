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

package apmhttp

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"go.elastic.co/apm"
)

const (
	// TraceparentHeader is the HTTP header for trace propagation.
	//
	// For backwards compatibility, this is currently an alias for
	// for ElasticTraceparentHeader, but the more specific constants
	// below should be preferred. In a future version this will be
	// replaced by the standard W3C header.
	TraceparentHeader = ElasticTraceparentHeader

	// ElasticTraceparentHeader is the legacy HTTP header for trace propagation,
	// maintained for backwards compatibility with older agents.
	ElasticTraceparentHeader = "Elastic-Apm-Traceparent"

	// W3CTraceparentHeader is the standard W3C Trace-Context HTTP
	// header for trace propagation.
	W3CTraceparentHeader = "Traceparent"

	// TracestateHeader is the standard W3C Trace-Context HTTP header
	// for vendor-specific trace propagation.
	TracestateHeader = "Tracestate"
)

// FormatTraceparentHeader formats the given trace context as a
// traceparent header.
func FormatTraceparentHeader(c apm.TraceContext) string {
	const version = 0
	return fmt.Sprintf("%02x-%032x-%016x-%02x", 0, c.Trace[:], c.Span[:], c.Options)
}

// ParseTraceparentHeader parses the given header, which is expected to be in
// the W3C Trace-Context traceparent format according to W3C Editor's Draft 23 May 2018:
//     https://w3c.github.io/trace-context/#traceparent-field
//
// Note that the returned TraceContext's Trace and Span fields are not necessarily
// valid. The caller must decide whether or not it wishes to disregard invalid
// trace/span IDs, and validate them as required using their provided Validate
// methods.
//
// The returned TraceContext's TraceState field will be the empty value. Use
// ParseTracestateHeader to parse that separately.
func ParseTraceparentHeader(h string) (apm.TraceContext, error) {
	var out apm.TraceContext
	if len(h) < 3 || h[2] != '-' {
		return out, errors.Errorf("invalid traceparent header %q", h)
	}
	var version byte
	if !strings.HasPrefix(h, "00") {
		decoded, err := hex.DecodeString(h[:2])
		if err != nil {
			return out, errors.Wrap(err, "error decoding traceparent header version")
		}
		version = decoded[0]
	}
	h = h[3:]

	switch version {
	case 255:
		// "Version 255 is invalid."
		return out, errors.Errorf("traceparent header version 255 is forbidden")
	default:
		// "If higher version is detected - implementation SHOULD try to parse it."
		fallthrough
	case 0:
		// Version 00:
		//
		//     version-format   = trace-id "-" span-id "-" trace-options
		//     trace-id         = 32HEXDIG
		//     span-id          = 16HEXDIG
		//     trace-options    = 2HEXDIG
		const (
			traceIDEnd        = 32
			spanIDStart       = traceIDEnd + 1
			spanIDEnd         = spanIDStart + 16
			traceOptionsStart = spanIDEnd + 1
			traceOptionsEnd   = traceOptionsStart + 2
		)
		switch {
		case len(h) < traceOptionsEnd,
			h[traceIDEnd] != '-',
			h[spanIDEnd] != '-',
			version == 0 && len(h) != traceOptionsEnd,
			version > 0 && len(h) > traceOptionsEnd && h[traceOptionsEnd] != '-':
			return out, errors.Errorf("invalid version %d traceparent header %q", version, h)
		}
		if _, err := hex.Decode(out.Trace[:], []byte(h[:traceIDEnd])); err != nil {
			return out, errors.Wrapf(err, "error decoding trace-id for version %d", version)
		}
		if err := out.Trace.Validate(); err != nil {
			return out, errors.Wrap(err, "invalid trace-id")
		}
		if _, err := hex.Decode(out.Span[:], []byte(h[spanIDStart:spanIDEnd])); err != nil {
			return out, errors.Wrapf(err, "error decoding span-id for version %d", version)
		}
		if err := out.Span.Validate(); err != nil {
			return out, errors.Wrap(err, "invalid span-id")
		}
		var traceOptions [1]byte
		if _, err := hex.Decode(traceOptions[:], []byte(h[traceOptionsStart:traceOptionsEnd])); err != nil {
			return out, errors.Wrapf(err, "error decoding trace-options for version %d", version)
		}
		out.Options = apm.TraceOptions(traceOptions[0])
		return out, nil
	}
}

// ParseTracestateHeader parses the given header, which is expected to be in the
// W3C Trace-Context tracestate format according to W3C Editor's Draft 18 Nov 2019:
//    https://w3c.github.io/trace-context/#tracestate-header
//
// Note that the returned TraceState is not necessarily valid. The caller must
// decide whether or not it wishes to disregard invalid tracestate entries, and
// validate them as required using their provided Validate methods.
//
// Multiple header values may be presented, in which case they will be treated as
// if they are concatenated together with commas.
func ParseTracestateHeader(h ...string) (apm.TraceState, error) {
	var entries []apm.TraceStateEntry
	for _, h := range h {
		for {
			h = strings.TrimSpace(h)
			if h == "" {
				break
			}
			kv := h
			if comma := strings.IndexRune(h, ','); comma != -1 {
				kv = strings.TrimSpace(h[:comma])
				h = h[comma+1:]
			} else {
				h = ""
			}
			equal := strings.IndexRune(kv, '=')
			if equal == -1 {
				return apm.TraceState{}, errors.New("missing '=' in tracestate entry")
			}
			entries = append(entries, apm.TraceStateEntry{Key: kv[:equal], Value: kv[equal+1:]})
		}
	}
	return apm.NewTraceState(entries...), nil
}
