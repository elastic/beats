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
// +build !integration

package multiline

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
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
	r, err = readfile.NewEncodeReader(ioutil.NopCloser(in), readfile.Config{
		Codec:      enc,
		BufferSize: 4096,
		Terminator: readfile.LineFeed,
	})
	if err != nil {
		t.Fatalf("Failed to initialize line reader: %v", err)
	}

	r, err = New(readfile.NewStripNewline(r, readfile.LineFeed), "\n", 1<<20, &cfg)
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
