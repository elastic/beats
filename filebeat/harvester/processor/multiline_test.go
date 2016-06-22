// +build !integration

package processor

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/stretchr/testify/assert"
)

type bufferSource struct{ buf *bytes.Buffer }

func (p bufferSource) Read(b []byte) (int, error) { return p.buf.Read(b) }
func (p bufferSource) Close() error               { return nil }
func (p bufferSource) Name() string               { return "buffer" }
func (p bufferSource) Stat() (os.FileInfo, error) { return nil, errors.New("unknown") }
func (p bufferSource) Continuable() bool          { return false }

func TestMultilineAfterOK(t *testing.T) {
	testMultilineOK(t,
		MultilineConfig{
			Pattern: regexp.MustCompile(`^[ \t] +`), // next line is indented by spaces
			Match:   "after",
		},
		"line1\n  line1.1\n  line1.2\n",
		"line2\n  line2.1\n  line2.2\n",
	)
}

func TestMultilineBeforeOK(t *testing.T) {
	testMultilineOK(t,
		MultilineConfig{
			Pattern: regexp.MustCompile(`\\$`), // previous line ends with \
			Match:   "before",
		},
		"line1 \\\nline1.1 \\\nline1.2\n",
		"line2 \\\nline2.1 \\\nline2.2\n",
	)
}

func TestMultilineAfterNegateOK(t *testing.T) {
	testMultilineOK(t,
		MultilineConfig{
			Pattern: regexp.MustCompile(`^-`), // first line starts with '-' at beginning of line
			Negate:  true,
			Match:   "after",
		},
		"-line1\n  - line1.1\n  - line1.2\n",
		"-line2\n  - line2.1\n  - line2.2\n",
	)
}

func TestMultilineBeforeNegateOK(t *testing.T) {
	testMultilineOK(t,
		MultilineConfig{
			Pattern: regexp.MustCompile(`;$`), // last line ends with ';'
			Negate:  true,
			Match:   "before",
		},
		"line1\nline1.1\nline1.2;\n",
		"line2\nline2.1\nline2.2;\n",
	)
}

func testMultilineOK(t *testing.T, cfg MultilineConfig, expected ...string) {
	_, buf := createLineBuffer(expected...)
	reader := createMultilineTestReader(t, buf, cfg)

	var lines []Line
	for {
		line, err := reader.Next()
		if err != nil {
			break
		}
		lines = append(lines, line)
	}

	if len(lines) != len(expected) {
		t.Fatalf("expected %v lines, read only %v line(s)", len(expected), len(lines))
	}

	for i, line := range lines {
		var tsZero time.Time

		assert.NotEqual(t, tsZero, line.Ts)
		assert.Equal(t, strings.TrimRight(expected[i], "\r\n "), string(line.Content))
		assert.Equal(t, len(expected[i]), int(line.Bytes))
	}
}

func createMultilineTestReader(t *testing.T, in *bytes.Buffer, cfg MultilineConfig) LineProcessor {
	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Fatalf("unable to find 'plain' encoding")
	}

	enc, err := encFactory(in)
	if err != nil {
		t.Fatalf("failed to initialize encoding: %v", err)
	}

	var reader LineProcessor
	reader, err = NewLineEncoder(in, enc, 4096)
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
