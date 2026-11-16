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

type wouldBlockCase struct {
	name     string
	cfg      Config
	lines    []string
	expected string
}

// wouldBlockCases are multiline configurations together with a set of lines
// that form a single event, which is completed only by the last line or by the
// end of the source.
func wouldBlockCases() []wouldBlockCase {
	afterPattern := match.MustCompile(`^[ \t]+`)
	beforePattern := match.MustCompile(`\\$`)
	negateAfter := match.MustCompile(`^-`)
	negateBefore := match.MustCompile(`;$`)
	flushStart := match.MustCompile(`EventStart`)
	flushEnd := match.MustCompile(`EventEnd`)
	whilePattern := match.MustCompile(`^{`)

	return []wouldBlockCase{
		{
			name:     "pattern match after",
			cfg:      Config{Type: patternMode, Pattern: &afterPattern, Match: "after"},
			lines:    []string{"line1", "  line1.1", "  line1.2"},
			expected: "line1\n  line1.1\n  line1.2",
		},
		{
			name:     "pattern match before",
			cfg:      Config{Type: patternMode, Pattern: &beforePattern, Match: "before"},
			lines:    []string{`line1 \`, `line1.1 \`, "line1.2"},
			expected: "line1 \\\nline1.1 \\\nline1.2",
		},
		{
			name:     "pattern match after negate",
			cfg:      Config{Type: patternMode, Pattern: &negateAfter, Match: "after", Negate: true},
			lines:    []string{"-line1", "  - line1.1", "  - line1.2"},
			expected: "-line1\n  - line1.1\n  - line1.2",
		},
		{
			name:     "pattern match before negate",
			cfg:      Config{Type: patternMode, Pattern: &negateBefore, Match: "before", Negate: true},
			lines:    []string{"line1", "line1.1", "line1.2;"},
			expected: "line1\nline1.1\nline1.2;",
		},
		{
			name: "pattern with flush_pattern",
			cfg: Config{
				Type: patternMode, Pattern: &flushStart, Match: "after", Negate: true,
				FlushPattern: &flushEnd,
			},
			lines:    []string{"EventStart", "EventId: 1", "EventEnd"},
			expected: "EventStart\nEventId: 1\nEventEnd",
		},
		{
			name:     "while_pattern",
			cfg:      Config{Type: whilePatternMode, Pattern: &whilePattern},
			lines:    []string{"{line1", "{line1.1", "{line1.2"},
			expected: "{line1\n{line1.1\n{line1.2",
		},
		{
			name:     "while_pattern negate",
			cfg:      Config{Type: whilePatternMode, Pattern: &whilePattern, Negate: true},
			lines:    []string{"panic:", "~stacktrace~", "~more~"},
			expected: "panic:\n~stacktrace~\n~more~",
		},
		{
			name:     "count",
			cfg:      Config{Type: countMode, LinesCount: 3},
			lines:    []string{"line1", "line2", "line3"},
			expected: "line1\nline2\nline3",
		},
	}
}

// TestMultilineWouldBlockPreservesBuffer verifies that ErrWouldBlock, even
// repeated after every line, neither flushes nor reorders the buffered lines:
// they are combined in one event once the remaining lines arrive.
func TestMultilineWouldBlockPreservesBuffer(t *testing.T) {
	for _, tc := range wouldBlockCases() {
		t.Run(tc.name, func(t *testing.T) {
			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")

			var steps []readertest.ScriptStep
			for _, l := range tc.lines {
				steps = append(steps, readertest.LineStep(l), readertest.WouldBlockStep, readertest.WouldBlockStep)
			}
			steps = append(steps, readertest.EOFStep)
			wantStalls := 2 * len(tc.lines)

			r, _ := newScriptedMultiline(t, tc.cfg, logger, steps...)

			var messages []reader.Message
			stalls := 0
			for {
				msg, err := r.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if errors.Is(err, reader.ErrWouldBlock) {
					assert.True(t, msg.IsEmpty(), "ErrWouldBlock must come with an empty message, got %+v", msg)
					stalls++
					require.LessOrEqual(t, stalls, wantStalls, "more ErrWouldBlock results than scripted")
					continue
				}
				require.NoError(t, err, "unexpected error from Next")
				messages = append(messages, msg)
			}

			assert.Equal(t, wantStalls, stalls, "every scripted ErrWouldBlock must be propagated")
			require.Len(t, messages, 1, "lines stalled by ErrWouldBlock must form a single event")
			assert.Equal(t, tc.expected, string(messages[0].Content), "event content")
			assert.Equal(t, linesBytes(tc.lines...), messages[0].Bytes, "event bytes")
			assert.True(t, hasMultilineFlag(t, messages[0]), "event must have the multiline flag")
			assert.Zero(t, logs.FilterMessage(timeoutFlushLog).Len(),
				"ErrWouldBlock must not be reported as a timeout flush")
		})
	}
}

// wouldBlockModes returns one case per concrete multiline reader
// implementation (patternReader, whilePatternReader, counterReader) for tests
// that exercise mode-dispatch or lifecycle paths independent of matcher flags.
func wouldBlockModes() []wouldBlockCase {
	afterPattern := match.MustCompile(`^[ \t]+`)
	whilePattern := match.MustCompile(`^{`)

	return []wouldBlockCase{
		{
			name:  "pattern",
			cfg:   Config{Type: patternMode, Pattern: &afterPattern, Match: "after"},
			lines: []string{"line1", "  line1.1"},
		},
		{
			name:  "while_pattern",
			cfg:   Config{Type: whilePatternMode, Pattern: &whilePattern},
			lines: []string{"{line1", "{line1.1"},
		},
		{
			name:  "count",
			cfg:   Config{Type: countMode, LinesCount: 3},
			lines: []string{"line1", "line2"},
		},
	}
}

// TestMultilineWouldBlockEmptyBuffer verifies that ErrWouldBlock and io.EOF
// with no buffered lines are passed through without a timeout flush log, and
// that the reader keeps working afterwards.
func TestMultilineWouldBlockEmptyBuffer(t *testing.T) {
	for _, tc := range wouldBlockModes() {
		t.Run(tc.name, func(t *testing.T) {
			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
			r, src := newScriptedMultiline(t, tc.cfg, logger, readertest.WouldBlockStep, readertest.WouldBlockStep)

			for i := range 2 {
				msg, err := r.Next()
				require.ErrorIs(t, err, reader.ErrWouldBlock, "stall %d with empty buffer", i)
				assert.True(t, msg.IsEmpty(), "stall %d: message must be empty", i)
			}

			pending, deadline := reader.PendingDeadline(r)
			assert.False(t, pending, "nothing is buffered")
			assert.True(t, deadline.IsZero(), "no deadline without buffered lines")

			// The reader is still usable.
			src.Push(readertest.LineStep(tc.lines[0]), readertest.EOFStep)
			msg, err := r.Next()
			require.NoError(t, err, "reading a line after ErrWouldBlock")
			assert.Equal(t, tc.lines[0], string(msg.Content), "line read after stalls")

			// io.EOF with an empty buffer.
			_, err = r.Next()
			require.ErrorIs(t, err, io.EOF, "io.EOF with an empty buffer")

			assert.Zero(t, logs.FilterMessage(timeoutFlushLog).Len(),
				"nothing was flushed, so no timeout flush may be logged")
		})
	}
}

func TestMultilinePendingDeadline(t *testing.T) {
	const timeout = 10 * time.Second
	zero := time.Duration(0)
	afterPattern := match.MustCompile(`^[ \t]+`)
	whilePattern := match.MustCompile(`^{`)

	tests := []struct {
		name         string
		cfg          Config
		lines        []string
		wantDeadline bool
	}{
		{
			name:         "pattern",
			cfg:          Config{Type: patternMode, Pattern: &afterPattern, Match: "after", Timeout: ptr(timeout)},
			lines:        []string{"line1", "  line1.1"},
			wantDeadline: true,
		},
		{
			name:         "while_pattern",
			cfg:          Config{Type: whilePatternMode, Pattern: &whilePattern, Timeout: ptr(timeout)},
			lines:        []string{"{line1", "{line1.1"},
			wantDeadline: true,
		},
		{
			name:  "pattern without timeout",
			cfg:   Config{Type: patternMode, Pattern: &afterPattern, Match: "after", Timeout: &zero},
			lines: []string{"line1", "  line1.1"},
		},
		{
			name:  "while_pattern without timeout",
			cfg:   Config{Type: whilePatternMode, Pattern: &whilePattern, Timeout: &zero},
			lines: []string{"{line1", "{line1.1"},
		},
		{
			name:  "count",
			cfg:   Config{Type: countMode, LinesCount: 3},
			lines: []string{"line1", "line2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, src := newScriptedMultiline(t, tc.cfg, logptest.NewTestingLogger(t, ""))

			pending, deadline := reader.PendingDeadline(r)
			assert.False(t, pending, "empty reader has nothing pending")
			assert.True(t, deadline.IsZero(), "empty reader has no deadline")

			before := time.Now()
			for _, l := range tc.lines {
				src.Push(readertest.LineStep(l))
			}
			src.Push(readertest.WouldBlockStep)
			_, err := r.Next()
			require.ErrorIs(t, err, reader.ErrWouldBlock, "stalling with buffered lines")
			after := time.Now()

			pending, deadline = reader.PendingDeadline(r)
			assert.True(t, pending, "buffered lines are pending")
			if tc.wantDeadline {
				assert.False(t, deadline.Before(before.Add(timeout)),
					"deadline %v must not precede the first line read at %v plus the timeout", deadline, before)
				assert.False(t, deadline.After(after.Add(timeout)),
					"deadline %v must not exceed the last line read at %v plus the timeout", deadline, after)
			} else {
				assert.True(t, deadline.IsZero(), "no timeout, so no deadline, got %v", deadline)
			}

			// A stall does not extend the deadline: it is based on the last line.
			time.Sleep(5 * time.Millisecond)
			src.Push(readertest.WouldBlockStep)
			_, err = r.Next()
			require.ErrorIs(t, err, reader.ErrWouldBlock, "second stall")
			_, again := reader.PendingDeadline(r)
			assert.Equal(t, deadline, again, "ErrWouldBlock must not move the deadline")

			msg, ok, err := reader.FlushPending(r, false)
			require.NoError(t, err, "flushing")
			require.True(t, ok, "flushing buffered lines")
			assert.NotEmpty(t, msg.Content, "flushed content")

			pending, deadline = reader.PendingDeadline(r)
			assert.False(t, pending, "flushed reader has nothing pending")
			assert.True(t, deadline.IsZero(), "flushed reader has no deadline")
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestMultilineFlushPending(t *testing.T) {
	afterPattern := match.MustCompile(`^[ \t]+`)
	whilePattern := match.MustCompile(`^{`)

	tests := []struct {
		name string
		cfg  Config
		// first and second events; each is left buffered by an ErrWouldBlock.
		first, second []string
		// logs is whether a timeout flush logs.
		logs bool
	}{
		{
			name:   "pattern",
			cfg:    Config{Type: patternMode, Pattern: &afterPattern, Match: "after"},
			first:  []string{"line1", "  line1.1", "  line1.2"},
			second: []string{"line2", "  line2.1"},
			logs:   true,
		},
		{
			name:   "while_pattern",
			cfg:    Config{Type: whilePatternMode, Pattern: &whilePattern},
			first:  []string{"{line1", "{line1.1", "{line1.2"},
			second: []string{"{line2", "{line2.1"},
			logs:   true,
		},
		{
			name:   "count",
			cfg:    Config{Type: countMode, LinesCount: 3},
			first:  []string{"line1", "line2"},
			second: []string{"line3", "line4"},
		},
	}

	for _, timedOut := range []bool{true, false} {
		for _, tc := range tests {
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
					require.True(t, ok, "FlushPending must flush the buffered lines")
					assert.Equal(t, strings.Join(lines, "\n"), string(msg.Content), "flushed content")
					assert.Equal(t, linesBytes(lines...), msg.Bytes, "flushed bytes")
					assert.Equal(t, len(lines) > 1, hasMultilineFlag(t, msg), "multiline flag of %d lines", len(lines))
					assert.False(t, msg.Ts.IsZero(), "flushed message timestamp")
				}

				stall(tc.first)
				flush(tc.first)
				pending, _ := reader.PendingDeadline(r)
				assert.False(t, pending, "nothing pending after a flush")

				// The reader restarted at readFirst: the next lines are a new event
				// that does not include the flushed ones.
				stall(tc.second)
				flush(tc.second)

				// A single buffered line is flushed without the multiline flag.
				stall(tc.second[:1])
				flush(tc.second[:1])

				// Nothing left to flush.
				msg, ok, err := reader.FlushPending(r, timedOut)
				require.NoError(t, err, "FlushPending with nothing buffered")
				assert.False(t, ok, "nothing was buffered")
				assert.True(t, msg.IsEmpty(), "no message when nothing was buffered")

				wantLogs := 0
				if timedOut && tc.logs {
					wantLogs = 3
				}
				assert.Equal(t, wantLogs, logs.FilterMessage(timeoutFlushLog).Len(),
					"timeout flush log count (timedOut=%t)", timedOut)
			})
		}
	}
}

// TestMultilineFlushPendingThenContinue verifies the reader keeps working after
// a flush, with lines resuming a new event as if the source had just started.
func TestMultilineFlushPendingThenContinue(t *testing.T) {
	pattern := match.MustCompile(`^[ \t]+`)
	cfg := Config{Type: patternMode, Pattern: &pattern, Match: "after"}
	r, src := newScriptedMultiline(t, cfg, logptest.NewTestingLogger(t, ""),
		readertest.LineStep("line1"), readertest.LineStep("  line1.1"), readertest.WouldBlockStep)

	_, err := r.Next()
	require.ErrorIs(t, err, reader.ErrWouldBlock, "stall")
	_, ok, err := reader.FlushPending(r, true)
	require.NoError(t, err, "flush")
	require.True(t, ok, "flush")

	// An indented line right after the flush must not be appended to the
	// flushed event; it starts a new one.
	src.Push(readertest.LineStep("  orphan"), readertest.LineStep("  orphan2"), readertest.LineStep("line3"), readertest.EOFStep)
	msg, err := r.Next()
	require.NoError(t, err, "next event")
	assert.Equal(t, "  orphan\n  orphan2", string(msg.Content), "event after flush")
	msg, err = r.Next()
	require.NoError(t, err, "last event")
	assert.Equal(t, "line3", string(msg.Content), "last event")
	_, err = r.Next()
	require.ErrorIs(t, err, io.EOF, "end of source")
}

func TestMultilineFlushPendingAfterClose(t *testing.T) {
	for _, tc := range wouldBlockModes() {
		t.Run(tc.name, func(t *testing.T) {
			r, src := newScriptedMultiline(t, tc.cfg, logptest.NewTestingLogger(t, ""))
			src.Push(readertest.LineStep(tc.lines[0]), readertest.LineStep(tc.lines[1]), readertest.WouldBlockStep)
			_, err := r.Next()
			require.ErrorIs(t, err, reader.ErrWouldBlock, "stall")
			pending, _ := reader.PendingDeadline(r)
			require.True(t, pending, "buffered lines before Close")

			require.NoError(t, r.Close(), "Close")

			pending, deadline := reader.PendingDeadline(r)
			assert.False(t, pending, "closed reader has nothing pending")
			assert.True(t, deadline.IsZero(), "closed reader has no deadline")
			_, ok, err := reader.FlushPending(r, false)
			require.NoError(t, err, "FlushPending after Close")
			assert.False(t, ok, "closed reader must not flush")
			_, err = r.Next()
			assert.ErrorIs(t, err, io.EOF, "Next after Close")
		})
	}
}
