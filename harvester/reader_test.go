package harvester

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang.org/x/text/transform"
)

// Sample texts are from http://www.columbia.edu/~kermit/utf8.html
var tests = []struct {
	encoding string
	strings  []string
}{
	{"plain", []string{"I can", "eat glass"}},
	{"latin1", []string{"I kå Glas frässa", "ond des macht mr nix!"}},
	{"utf-16be", []string{"Pot să mănânc sticlă", "și ea nu mă rănește."}},
	{"utf-16le", []string{"काचं शक्नोम्यत्तुम् ।", "नोपहिनस्ति माम् ॥"}},
	{"big5", []string{"我能吞下玻", "璃而不傷身體。"}},
	{"gb18030", []string{"我能吞下玻璃", "而不傷身。體"}},
	{"euc-kr", []string{" 나는 유리를 먹을 수 있어요.", " 그래도 아프지 않아요"}},
	{"euc-jp", []string{"私はガラスを食べられます。", "それは私を傷つけません。"}},
}

func TestReaderEncodings(t *testing.T) {
	for _, test := range tests {
		t.Logf("test codec: %v", test.encoding)

		codec, ok := findEncoding(test.encoding)
		if !ok {
			t.Errorf("can not find encoding '%v'", test.encoding)
			continue
		}

		buffer := bytes.NewBuffer(nil)

		// write with encoding to buffer
		writer := transform.NewWriter(buffer, codec.NewEncoder())
		var expectedCount []int
		for _, line := range test.strings {
			writer.Write([]byte(line))
			writer.Write([]byte{'\n'})
			expectedCount = append(expectedCount, buffer.Len())
		}

		// create line reader
		reader, err := newLineReader(buffer, codec, 1024)
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}

		// read decodec lines from buffer
		var readLines []string
		var byteCounts []int
		current := 0
		for {
			bytes, sz, err := reader.next()
			if sz > 0 {
				readLines = append(readLines, string(bytes[:len(bytes)-1]))
			}

			if err != nil {
				break
			}

			current += sz
			byteCounts = append(byteCounts, current)
		}

		// validate lines and byte offsets
		if len(test.strings) != len(readLines) {
			t.Errorf("number of lines mismatch (expected=%v actual=%v)",
				len(test.strings), len(readLines))
			continue
		}
		for i := range test.strings {
			expected := test.strings[i]
			actual := readLines[i]
			assert.Equal(t, expected, actual)
			assert.Equal(t, expectedCount[i], byteCounts[i])
		}
	}
}

func TestReaderPartialWithEncodings(t *testing.T) {
	for _, test := range tests {
		t.Logf("test codec: %v", test.encoding)

		codec, ok := findEncoding(test.encoding)
		if !ok {
			t.Errorf("can not find encoding '%v'", test.encoding)
			continue
		}

		buffer := bytes.NewBuffer(nil)
		writer := transform.NewWriter(buffer, codec.NewEncoder())
		reader, err := newLineReader(buffer, codec, 1024)
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}

		var expected []string
		var partials []string
		lastString := ""
		for _, str := range test.strings {
			writer.Write([]byte(str))
			lastString += str
			expected = append(expected, lastString)

			line, sz, err := reader.next()
			assert.NotNil(t, err)
			assert.Equal(t, 0, sz)
			assert.Nil(t, line)

			partial, _, err := reader.partial()
			partials = append(partials, string(partial))
			t.Logf("partials: %v", partials)
		}

		// finish line:
		writer.Write([]byte{'\n'})

		// finally read line
		line, _, err := reader.next()
		assert.Nil(t, err)
		t.Logf("line: '%v'", line)

		// validate partial lines
		if len(test.strings) != len(expected) {
			t.Errorf("number of lines mismatch (expected=%v actual=%v)",
				len(test.strings), len(partials))
			continue
		}
		for i := range expected {
			assert.Equal(t, expected[i], partials[i])
		}

		assert.Equal(t, lastString+"\n", string(line))
	}
}
