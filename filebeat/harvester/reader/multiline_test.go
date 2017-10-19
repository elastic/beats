// +build !integration

package reader

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/libbeat/common/match"
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
		MultilineConfig{
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
		MultilineConfig{
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
		MultilineConfig{
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
		MultilineConfig{
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
		MultilineConfig{
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
		MultilineConfig{
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
		MultilineConfig{
			Pattern: &pattern,
			Negate:  true,
			Match:   "before",
		},
		2,
		"line1\n\n\nline1.2;\n",
		"line2\nline2.1\nline2.2;\n",
	)
}

func testMultilineOK(t *testing.T, cfg MultilineConfig, events int, expected ...string) {
	_, buf := createLineBuffer(expected...)
	reader := createMultilineTestReader(t, buf, cfg)

	var messages []Message
	for {
		message, err := reader.Next()
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

func createMultilineTestReader(t *testing.T, in *bytes.Buffer, cfg MultilineConfig) Reader {
	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Fatalf("unable to find 'plain' encoding")
	}

	enc, err := encFactory(in)
	if err != nil {
		t.Fatalf("failed to initialize encoding: %v", err)
	}

	var reader Reader
	reader, err = NewEncode(in, enc, 4096)
	if err != nil {
		t.Fatalf("Failed to initialize line reader: %v", err)
	}

	reader, err = NewMultiline(NewStripNewline(reader), "\n", 1<<20, &cfg)
	if err != nil {
		t.Fatalf("failed to initializ reader: %v", err)
	}

	return reader
}

func createLineBuffer(lines ...string) ([]string, *bytes.Buffer) {
	buf := bytes.NewBuffer(nil)
	for _, line := range lines {
		buf.WriteString(line)
	}
	return lines, buf
}
