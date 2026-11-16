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

//go:build !integration

package multiline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readertest"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

type bufferSource struct{ buf *bytes.Buffer }

func (p bufferSource) Read(b []byte) (int, error) { return p.buf.Read(b) }
func (p bufferSource) Close() error               { return nil }
func (p bufferSource) Name() string               { return "buffer" }
func (p bufferSource) Stat() (os.FileInfo, error) { return nil, errors.New("unknown") }
func (p bufferSource) Continuable() bool          { return false }

func TestMultilineAfterOK(t *testing.T) {
	pattern := match.MustCompile(`^[ \t] +`) // next line is indented by spaces
	testMultilineOK(t,
		Config{
			Type:    patternMode,
			Pattern: &pattern,
			Match:   "after",
		},
		2,
		"line1\n  line1.1\n  line1.2\n",
		"line2\n  line2.1\n  line2.2\n",
	)
}

func TestMultilineBeforeOK(t *testing.T) {
	pattern := match.MustCompile(`\\$`) // previous line ends with \

	testMultilineOK(t,
		Config{
			Type:    patternMode,
			Pattern: &pattern,
			Match:   "before",
		},
		2,
		"line1 \\\nline1.1 \\\nline1.2\n",
		"line2 \\\nline2.1 \\\nline2.2\n",
	)
}

func TestMultilineAfterNegateOK(t *testing.T) {
	pattern := match.MustCompile(`^-`) // first line starts with '-' at beginning of line

	testMultilineOK(t,
		Config{
			Type:    patternMode,
			Pattern: &pattern,
			Negate:  true,
			Match:   "after",
		},
		2,
		"-line1\n  - line1.1\n  - line1.2\n",
		"-line2\n  - line2.1\n  - line2.2\n",
	)
}

func TestMultilineBeforeNegateOK(t *testing.T) {
	pattern := match.MustCompile(`;$`) // last line ends with ';'

	testMultilineOK(t,
		Config{
			Type:    patternMode,
			Pattern: &pattern,
			Negate:  true,
			Match:   "before",
		},
		2,
		"line1\nline1.1\nline1.2;\n",
		"line2\nline2.1\nline2.2;\n",
	)
}

func TestMultilineAfterNegateOKFlushPattern(t *testing.T) {
	flushMatcher := match.MustCompile(`EventEnd`)
	pattern := match.MustCompile(`EventStart`)

	testMultilineOK(t,
		Config{
			Type:         patternMode,
			Pattern:      &pattern,
			Negate:       true,
			Match:        "after",
			FlushPattern: &flushMatcher,
		},
		3,
		"EventStart\nEventId: 1\nEventEnd\n",
		"OtherThingInBetween\n", // this should be a separate event..
		"EventStart\nEventId: 2\nEventEnd\n",
	)
}

func TestMultilineAfterNegateOKFlushPatternWhereTheFirstLinesDosentMatchTheStartPattern(t *testing.T) {
	flushMatcher := match.MustCompile(`EventEnd`)
	pattern := match.MustCompile(`EventStart`)

	testMultilineOK(t,
		Config{
			Type:         patternMode,
			Pattern:      &pattern,
			Negate:       true,
			Match:        "after",
			FlushPattern: &flushMatcher,
		},
		3, //first two non-matching lines, will be merged to one event
		"StartLineThatDosentMatchTheEvent\nOtherThingInBetween\n",
		"EventStart\nEventId: 2\nEventEnd\n",
		"EventStart\nEventId: 3\nEventEnd\n",
	)
}

func TestMultilineBeforeNegateOKWithEmptyLine(t *testing.T) {
	pattern := match.MustCompile(`;$`) // last line ends with ';'
	testMultilineOK(t,
		Config{
			Type:    patternMode,
			Pattern: &pattern,
			Negate:  true,
			Match:   "before",
		},
		2,
		"line1\n\n\nline1.2;\n",
		"line2\nline2.1\nline2.2;\n",
	)
}

func TestMultilineAfterTruncated(t *testing.T) {
	pattern := match.MustCompile(`^[ ]`) // next line is indented a space
	maxLines := 2
	testMultilineTruncated(t,
		Config{
			Type:     patternMode,
			Pattern:  &pattern,
			Match:    "after",
			MaxLines: &maxLines,
		},
		2,
		true,
		[]string{
			"line1\n line1.1\n line1.2\n",
			"line2\n line2.1\n line2.2\n"},
		[]string{
			"line1\n line1.1",
			"line2\n line2.1"},
	)
	testMultilineTruncated(t,
		Config{
			Type:     patternMode,
			Pattern:  &pattern,
			Match:    "after",
			MaxLines: &maxLines,
		},
		2,
		false,
		[]string{
			"line1\n line1.1\n",
			"line2\n line2.1\n"},
		[]string{
			"line1\n line1.1",
			"line2\n line2.1"},
	)
}

func TestMultilineCount(t *testing.T) {
	maxLines := 2
	testMultilineOK(t,
		Config{
			Type:       countMode,
			MaxLines:   &maxLines,
			LinesCount: 2,
		},
		2,
		"line1\n line1.1\n",
		"line2\n line2.1\n",
	)
	maxLines = 4
	testMultilineOK(t,
		Config{
			Type:       countMode,
			MaxLines:   &maxLines,
			LinesCount: 4,
		},
		2,
		"line1\n line1.1\nline2\n line2.1\n",
		"line3\n line3.1\nline4\n line4.1\n",
	)
	maxLines = 1
	testMultilineOK(t,
		Config{
			Type:       countMode,
			MaxLines:   &maxLines,
			LinesCount: 1,
		},
		8,
		"line1\n", "line1.1\n", "line2\n", "line2.1\n", "line3\n", "line3.1\n", "line4\n", "line4.1\n",
	)
	maxLines = 2
	testMultilineTruncated(t,
		Config{
			Type:       countMode,
			MaxLines:   &maxLines,
			LinesCount: 3,
		},
		4,
		true,
		[]string{"line1\n line1.1\n line1.2\n", "line2\n line2.1\n line2.2\n", "line3\n line3.1\n line3.2\n", "line4\n line4.1\n line4.3\n"},
		[]string{"line1\n", "line2\n", "line3\n", "line4\n"},
	)
}

func TestMultilineWhilePattern(t *testing.T) {
	pattern := match.MustCompile(`^{`)
	testMultilineOK(t,
		Config{
			Type:    whilePatternMode,
			Pattern: &pattern,
			Negate:  false,
		},
		3,
		"{line1\n{line1.1\n",
		"not matched line\n",
		"{line2\n{line2.1\n",
	)
	// use negated
	testMultilineOK(t,
		Config{
			Type:    whilePatternMode,
			Pattern: &pattern,
			Negate:  true,
		},
		3,
		"{line1\n",
		"panic:\n~stacktrace~\n",
		"{line2\n",
	)
	// truncated
	maxLines := 2
	testMultilineTruncated(t,
		Config{
			Type:     whilePatternMode,
			Pattern:  &pattern,
			MaxLines: &maxLines,
		},
		1,
		true,
		[]string{
			"{line1\n{line1.1\n{line1.2\n"},
		[]string{
			"{line1\n{line1.1\n"},
	)
}

func testMultilineOK(t *testing.T, cfg Config, events int, expected ...string) {
	_, buf := createLineBuffer(expected...)
	r := createMultilineTestReader(t, buf, cfg)

	var messages []reader.Message
	for {
		message, err := r.Next()
		if err != nil {
			break
		}

		messages = append(messages, message)
	}

	if len(messages) != events {
		t.Fatalf("expected %v lines, read only %v line(s)", len(expected), len(messages))
	}

	for i, message := range messages {
		var tsZero time.Time

		assert.NotEqual(t, tsZero, message.Ts)
		assert.Equal(t, strings.TrimRight(expected[i], "\r\n "), string(message.Content))
		assert.Equal(t, len(expected[i]), int(message.Bytes))
	}
}

func testMultilineTruncated(t *testing.T, cfg Config, events int, truncated bool, input, expected []string) {
	_, buf := createLineBuffer(input...)
	r := createMultilineTestReader(t, buf, cfg)

	var messages []reader.Message
	for {
		message, err := r.Next()
		if err != nil {
			break
		}

		messages = append(messages, message)
	}

	if len(messages) != events {
		t.Fatalf("expected %v lines, read only %v line(s)", len(expected), len(messages))
	}

	for _, message := range messages {
		found := false
		multiline := false
		statusFlags, err := message.Fields.GetValue("log.flags")
		if err != nil {
			if !truncated {
				assert.False(t, found)
				return
			}
			t.Fatalf("error while getting log.status field: %v", err)
		}

		switch flags := statusFlags.(type) {
		case []string:
			for _, f := range flags {
				if f == "truncated" {
					found = true
				}
				if f == "multiline" {
					multiline = true
				}
			}
		default:
			t.Fatalf("incorrect type for log.flags")
		}

		if truncated {
			assert.True(t, found)
		} else {
			assert.False(t, found)
		}
		assert.True(t, multiline)
	}
}

func createMultilineTestReader(t *testing.T, in *bytes.Buffer, cfg Config) reader.Reader {
	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Fatalf("unable to find 'plain' encoding")
	}

	enc, err := encFactory(in)
	if err != nil {
		t.Fatalf("failed to initialize encoding: %v", err)
	}

	var r reader.Reader
	r, err = readfile.NewEncodeReader(io.NopCloser(in), readfile.Config{
		Codec:      enc,
		BufferSize: 4096,
		Terminator: readfile.LineFeed,
	}, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatalf("Failed to initialize line reader: %v", err)
	}

	r, err = New(readfile.NewStripNewline(r, readfile.LineFeed), "\n", 1<<20, &cfg, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatalf("failed to initialize reader: %v", err)
	}

	return r
}

func createLineBuffer(lines ...string) ([]string, *bytes.Buffer) {
	buf := bytes.NewBuffer(nil)
	for _, line := range lines {
		buf.WriteString(line)
	}
	return lines, buf
}

const timeoutFlushLog = "Multiline event flushed because timeout reached."

func newScriptedMultiline(t *testing.T, cfg Config, logger *logp.Logger, steps ...readertest.ScriptStep) (reader.Reader, *readertest.ScriptedReader) {
	t.Helper()
	src := readertest.NewScriptedReader(steps...)
	r, err := New(src, "\n", 1<<20, &cfg, logger)
	require.NoError(t, err, "creating multiline reader")
	return r, src
}

func hasMultilineFlag(t *testing.T, msg reader.Message) bool {
	t.Helper()
	flags, err := msg.Fields.GetValue("log.flags")
	if err != nil {
		return false
	}
	list, ok := flags.([]string)
	require.True(t, ok, "log.flags must be a []string, got %T", flags)
	return slices.Contains(list, "multiline")
}

func linesBytes(lines ...string) int {
	n := 0
	for _, l := range lines {
		n += len(l) + 1
	}
	return n
}

type wouldBlockMode struct {
	name       string
	cfg        Config
	lines      []string
	second     []string
	hasTimeout bool
}

// wouldBlockModes returns one case per concrete multiline reader
// implementation (patternReader, whilePatternReader, counterReader) for tests
// that exercise ErrWouldBlock, PendingDeadline, or FlushPending.
func wouldBlockModes() []wouldBlockMode {
	afterPattern := match.MustCompile(`^[ \t]+`)
	whilePattern := match.MustCompile(`^{`)

	return []wouldBlockMode{
		{
			name:       "pattern",
			cfg:        Config{Type: patternMode, Pattern: &afterPattern, Match: "after"},
			lines:      []string{"line1", "  line1.1", "  line1.2"},
			second:     []string{"  line2", "  line2.1"},
			hasTimeout: true,
		},
		{
			name:       "while_pattern",
			cfg:        Config{Type: whilePatternMode, Pattern: &whilePattern},
			lines:      []string{"{line1", "{line1.1", "{line1.2"},
			second:     []string{"{line2", "{line2.1"},
			hasTimeout: true,
		},
		{
			name:   "count",
			cfg:    Config{Type: countMode, LinesCount: 3},
			lines:  []string{"line1", "line2", "line3"},
			second: []string{"line4", "line5", "line6"},
		},
	}
}

// TestMultilineWouldBlockPreservesBuffer verifies that ErrWouldBlock on an
// empty buffer passes through cleanly and, when repeated after buffered lines,
// preserves them until the event completes.
func TestMultilineWouldBlockPreservesBuffer(t *testing.T) {
	for _, tc := range wouldBlockModes() {
		t.Run(tc.name, func(t *testing.T) {
			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
			r, src := newScriptedMultiline(
				t, tc.cfg, logger, readertest.WouldBlockStep, readertest.WouldBlockStep,
			)

			for i := range 2 {
				msg, err := r.Next()
				require.ErrorIs(t, err, reader.ErrWouldBlock, "empty-buffer stall %d", i)
				assert.True(t, msg.IsEmpty(), "empty-buffer stall %d: empty msg", i)
			}
			pending, deadline := reader.PendingDeadline(r)
			assert.False(t, pending, "nothing pending on empty buffer")
			assert.True(t, deadline.IsZero(), "no deadline on empty buffer")

			for _, l := range tc.lines {
				src.Push(readertest.LineStep(l), readertest.WouldBlockStep, readertest.WouldBlockStep)
			}
			src.Push(readertest.EOFStep)
			wantStalls := 2 * len(tc.lines)

			var messages []reader.Message
			stalls := 0
			for {
				msg, err := r.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if errors.Is(err, reader.ErrWouldBlock) {
					assert.True(t, msg.IsEmpty(), "ErrWouldBlock must return empty message")
					stalls++
					require.LessOrEqual(t, stalls, wantStalls, "unexpected ErrWouldBlock")
					continue
				}
				require.NoError(t, err, "unexpected error from Next")
				messages = append(messages, msg)
			}

			assert.Equal(t, wantStalls, stalls, "scripted ErrWouldBlock count")
			require.Len(t, messages, 1, "stalled lines must form one event")
			assert.Equal(t, strings.Join(tc.lines, "\n"), string(messages[0].Content), "event content")
			assert.Equal(t, linesBytes(tc.lines...), messages[0].Bytes, "event bytes")
			assert.True(t, hasMultilineFlag(t, messages[0]), "event must have multiline flag")
			assert.Zero(t, logs.FilterMessage(timeoutFlushLog).Len(), "no timeout flush on ErrWouldBlock")
		})
	}
}

func TestMultilinePendingDeadline(t *testing.T) {
	timeout := 10 * time.Second
	zero := time.Duration(0)

	for _, tc := range wouldBlockModes() {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.Timeout = &timeout
			r, src := newScriptedMultiline(t, cfg, logptest.NewTestingLogger(t, ""))

			before := time.Now()
			for _, l := range tc.lines[:2] {
				src.Push(readertest.LineStep(l))
			}
			src.Push(readertest.WouldBlockStep)
			_, err := r.Next()
			require.ErrorIs(t, err, reader.ErrWouldBlock, "stalling with buffered lines")
			after := time.Now()

			pending, deadline := reader.PendingDeadline(r)
			assert.True(t, pending, "buffered lines are pending")
			if tc.hasTimeout {
				assert.False(t, deadline.Before(before.Add(timeout)), "deadline before lower bound")
				assert.False(t, deadline.After(after.Add(timeout)), "deadline after upper bound")
			} else {
				assert.True(t, deadline.IsZero(), "count mode has no deadline")
			}

			// A stall does not extend the deadline: it is based on the last line.
			time.Sleep(5 * time.Millisecond)
			src.Push(readertest.WouldBlockStep)
			_, err = r.Next()
			require.ErrorIs(t, err, reader.ErrWouldBlock, "second stall")
			_, again := reader.PendingDeadline(r)
			assert.Equal(t, deadline, again, "ErrWouldBlock must not move deadline")

			if tc.hasTimeout {
				cfg.Timeout = &zero
				r, src = newScriptedMultiline(t, cfg, logptest.NewTestingLogger(t, ""))
				src.Push(readertest.LineStep(tc.lines[0]), readertest.WouldBlockStep)
				_, err = r.Next()
				require.ErrorIs(t, err, reader.ErrWouldBlock, "stalling without timeout")
				pending, deadline = reader.PendingDeadline(r)
				assert.True(t, pending, "buffered line is pending")
				assert.True(t, deadline.IsZero(), "zero timeout has no deadline")
			}
		})
	}
}

func TestMultilineFlushPending(t *testing.T) {
	for _, timedOut := range []bool{true, false} {
		for _, tc := range wouldBlockModes() {
			t.Run(fmt.Sprintf("%s timedOut=%t", tc.name, timedOut), func(t *testing.T) {
				logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
				r, src := newScriptedMultiline(t, tc.cfg, logger)

				stall := func(lines []string) {
					t.Helper()
					for _, l := range lines {
						src.Push(readertest.LineStep(l))
					}
					src.Push(readertest.WouldBlockStep)
					_, err := r.Next()
					require.ErrorIs(t, err, reader.ErrWouldBlock, "stalling after %v", lines)
				}
				flush := func(lines []string) {
					t.Helper()
					msg, ok, err := reader.FlushPending(r, timedOut)
					require.NoError(t, err, "FlushPending")
					require.True(t, ok, "FlushPending must flush buffered lines")
					assert.Equal(t, strings.Join(lines, "\n"), string(msg.Content), "flushed content")
					assert.Equal(t, linesBytes(lines...), msg.Bytes, "flushed bytes")
					assert.Equal(t, len(lines) > 1, hasMultilineFlag(t, msg), "multiline flag")
					assert.False(t, msg.Ts.IsZero(), "flushed message timestamp")
				}

				stall(tc.lines[:2])
				flush(tc.lines[:2])
				pending, deadline := reader.PendingDeadline(r)
				assert.False(t, pending, "nothing pending after flush")
				assert.True(t, deadline.IsZero(), "no deadline after flush")

				// A single buffered line is flushed without the multiline flag.
				stall(tc.lines[:1])
				flush(tc.lines[:1])

				// Nothing left to flush.
				msg, ok, err := reader.FlushPending(r, timedOut)
				require.NoError(t, err, "FlushPending with empty buffer")
				assert.False(t, ok, "nothing buffered")
				assert.True(t, msg.IsEmpty(), "empty message when nothing buffered")

				// Reader restarted at readFirst: subsequent lines form a new event.
				for _, l := range tc.second {
					src.Push(readertest.LineStep(l))
				}
				src.Push(readertest.EOFStep)
				msg, err = r.Next()
				require.NoError(t, err, "Next after FlushPending")
				assert.Equal(t, strings.Join(tc.second, "\n"), string(msg.Content), "event after flush")
				_, err = r.Next()
				require.ErrorIs(t, err, io.EOF, "end of source")

				wantLogs := 0
				if timedOut && tc.hasTimeout {
					wantLogs = 2
				}
				assert.Equal(t, wantLogs, logs.FilterMessage(timeoutFlushLog).Len(), "timeout flush logs")

				// Closing discards buffered lines and prevents later flushes.
				stall(tc.lines[:2])
				require.NoError(t, r.Close(), "Close")
				pending, deadline = reader.PendingDeadline(r)
				assert.False(t, pending, "closed reader has nothing pending")
				assert.True(t, deadline.IsZero(), "closed reader has no deadline")
				_, ok, err = reader.FlushPending(r, timedOut)
				require.NoError(t, err, "FlushPending after Close")
				assert.False(t, ok, "closed reader must not flush")
				_, err = r.Next()
				assert.ErrorIs(t, err, io.EOF, "Next after Close")
			})
		}
	}
}
