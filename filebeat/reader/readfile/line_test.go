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

// +build !integration

package readfile

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/filebeat/reader/readfile/encoding"
)

// Sample texts are from http://www.columbia.edu/~kermit/utf8.html
var tests = []struct {
	encoding string
	strings  []string
}{
	{"plain", []string{""}},
	{"plain", []string{"", ""}},
	{"plain", []string{"I can"}},
	{"plain", []string{"I can", "eat glass"}},

	{"latin1", []string{""}},
	{"latin1", []string{"I can"}},
	{"latin1", []string{"I kå Glas frässa"}},
	{"latin1", []string{"I kå Glas frässa", "ond des macht mr nix!"}},

	{"utf-8", []string{""}},
	{"utf-8", []string{"I can"}},
	{"utf-8", []string{"árvíztűrő tükörfúrógép"}},
	{"utf-8", []string{"A nap tüze, látod,", "a fürge diákot", "a hegyre kicsalta: a csúcsra kiállt."}},

	{"utf-16", []string{""}},
	{"utf-16", []string{"I can"}},
	{"utf-16", []string{"I can", "eat glass"}},
	{"utf-16", []string{"Pot să mănânc sticlă"}},
	{"utf-16", []string{"Pot să mănânc sticlă", "și ea nu mă rănește."}},
	{"utf-16", []string{"\u0001234", "\u0000", "\u3453"}},

	{"utf-16be", []string{""}},
	{"utf-16be", []string{"I can"}},
	{"utf-16be", []string{"I can", "eat glass"}},
	{"utf-16be", []string{"Pot să mănânc sticlă"}},
	{"utf-16be", []string{"Pot să mănânc sticlă", "și ea nu mă rănește."}},

	{"utf-16le", []string{""}},
	{"utf-16le", []string{"I can"}},
	{"utf-16le", []string{"I can", "eat glass"}},
	{"utf-16le", []string{"काचं शक्नोम्यत्तुम् ।"}},
	{"utf-16le", []string{"काचं शक्नोम्यत्तुम् ।", "नोपहिनस्ति माम् ॥"}},

	{"big5", []string{"我能吞下玻", "璃而不傷身體。"}},
	{"gb18030", []string{"我能吞下玻璃", "而不傷身。體"}},
	{"euc-kr", []string{" 나는 유리를 먹을 수 있어요.", " 그래도 아프지 않아요"}},
	{"euc-jp", []string{"私はガラスを食べられます。", "それは私を傷つけません。"}},
}

var testsWithOffset = []struct {
	encoding string
	input    []string
	state    State
	expected []string
}{
	{
		"plain",
		[]string{"to read", "to read"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 0,
			ConvertedStreamOffset:  0,
		},
		[]string{"to read", "to read"},
	},
	{
		"plain",
		[]string{"already encountered", "already encountered", "to read"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 40,
			ConvertedStreamOffset:  40,
		},
		[]string{"to read"},
	},
	{
		"plain",
		[]string{"already encountered", "to read", "to read"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 20,
			ConvertedStreamOffset:  20,
		},
		[]string{"to read", "to read"},
	},
	{
		"plain",
		[]string{"already encountered", "already encountered", "to read"},
		State{
			EncodedOffset:          20,
			ConvertedSegmentOffset: 20,
			ConvertedStreamOffset:  40,
		},
		[]string{"to read"},
	},
	{
		"plain",
		[]string{"already encountered", "already encountered", "to read"},
		State{
			EncodedOffset:          24,
			ConvertedSegmentOffset: 16,
			ConvertedStreamOffset:  40,
		},
		[]string{"to read"},
	},
	{
		"latin1",
		[]string{"to réad", "to réad"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 0,
			ConvertedStreamOffset:  0,
		},
		[]string{"to réad", "to réad"},
	},
	{
		"latin1",
		[]string{"alréady éncountered", "alréady éncountered", "to réad"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 44,
			ConvertedStreamOffset:  44,
		},
		[]string{"to réad"},
	},
	{
		"latin1",
		[]string{"alréady éncountered", "to réad", "to réad"},
		State{
			EncodedOffset:          0,
			ConvertedSegmentOffset: 22,
			ConvertedStreamOffset:  22,
		},
		[]string{"to réad", "to réad"},
	},
	{
		"latin1",
		[]string{"alréady éncountered", "alréady éncountered", "to réad"},
		State{
			EncodedOffset:          20,
			ConvertedSegmentOffset: 22,
			ConvertedStreamOffset:  44,
		},
		[]string{"to réad"},
	},
	{
		"utf-16",
		[]string{"alréady éncountered", "alréady éncountered", "to réad"},
		State{
			EncodedOffset:          40,
			ConvertedSegmentOffset: 22,
			ConvertedStreamOffset:  44,
		},
		[]string{"to réad"},
	},
	{
		"utf-16",
		[]string{"alréady éncountered", "alréady éncountered", "to réad"},
		State{
			EncodedOffset:          30,
			ConvertedSegmentOffset: 27,
			ConvertedStreamOffset:  44,
		},
		[]string{"to réad"},
	},
}

func TestReaderEncodings(t *testing.T) {
	testReaderWithEncodings(t, 1024)
}

func TestReaderEncodingsWithSmallBuffer(t *testing.T) {
	testReaderWithEncodings(t, 3)
}

func testReaderWithEncodings(t *testing.T, bufferSize int) {
	for _, test := range tests {
		t.Logf("test codec: %v", test.encoding)

		reader, expectedCount, err := newTestReader(test.encoding, test.strings, bufferSize)
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}

		// read decodec lines from buffer
		var readLines []string
		var byteCounts []int
		for {
			bytes, sz, err := reader.Next()
			if sz > 0 {
				readLines = append(readLines, string(bytes[:len(bytes)-1]))
			}

			if err != nil {
				break
			}

			byteCounts = append(byteCounts, sz)
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

func TestReaderWithOffset(t *testing.T) {
	for _, test := range testsWithOffset {
		t.Logf("test codec: %v", test.encoding)

		reader, expectedCount, err := newTestReader(test.encoding, test.input, 1024)
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}
		err = reader.SetState(test.state)
		if err != nil {
			t.Errorf("failed to set the state of reader: %v", err)
			continue
		}

		// read decodec lines from buffer
		var readLines []string
		var byteCounts []int
		for {
			bytes, sz, err := reader.Next()
			if sz > 0 {
				readLines = append(readLines, string(bytes[:len(bytes)-1]))
			}

			if err != nil {
				break
			}

			byteCounts = append(byteCounts, sz)
		}

		// validate lines and byte offsets
		if len(test.expected) != len(readLines) {
			t.Errorf("number of lines mismatch (expected=%v actual=%v)",
				len(test.expected), len(readLines))
			continue
		}
		for i := range test.expected {
			expected := test.expected[i]
			actual := readLines[i]
			assert.Equal(t, expected, actual)
			assert.Equal(t, expectedCount[len(test.input)-1-i], byteCounts[i])
		}
	}

}

func newTestReader(enc string, input []string, bufferSize int) (*LineReader, []int, error) {
	codecFactory, ok := encoding.FindEncoding(enc)
	if !ok {
		return nil, nil, fmt.Errorf("can not find encoding '%v'", enc)
	}

	buffer := bytes.NewBuffer(nil)
	codec, _ := codecFactory(buffer)

	// write with encoding to buffer
	writer := transform.NewWriter(buffer, codec.NewEncoder())
	var expectedCount []int
	for _, line := range input {
		writer.Write([]byte(line))
		writer.Write([]byte{'\n'})
		expectedCount = append(expectedCount, len(line)+1)
	}

	// create line reader
	config := Config{
		Codec:      codec,
		Separator:  []byte("\n"),
		BufferSize: 1024,
	}
	r, err := NewLineReader(buffer, config)
	return r, expectedCount, err
}

func TestReadSingleLongLine(t *testing.T) {
	testReadLineLengths(t, []int{10 * 1024})
}

func TestReadIncreasingLineLengths(t *testing.T) {
	lineLengths := []int{200, 400, 800, 1000, 2048, 4069}
	testReadLineLengths(t, lineLengths)
}

func TestReadDecreasingLineLengths(t *testing.T) {
	lineLengths := []int{4096, 2048, 1000, 800, 400, 200}
	testReadLineLengths(t, lineLengths)
}

func TestReadRandomLineLengths(t *testing.T) {
	minLength := 100
	maxLength := 80000
	numLines := 100

	lineLengths := make([]int, numLines)
	for i := 0; i < numLines; i++ {
		lineLengths[i] = rand.Intn(maxLength-minLength) + minLength
	}

	testReadLineLengths(t, lineLengths)
}

func testReadLineLengths(t *testing.T, lineLengths []int) {
	// create lines + stream buffer
	var lines [][]byte
	for _, lineLength := range lineLengths {
		inputLine := make([]byte, lineLength+1)
		for i := 0; i < lineLength; i++ {
			char := rand.Intn('z'-'A') + 'A'
			inputLine[i] = byte(char)
		}
		inputLine[len(inputLine)-1] = '\n'
		lines = append(lines, inputLine)
	}

	testReadLines(t, lines)
}

func testReadLines(t *testing.T, inputLines [][]byte) {
	var inputStream []byte
	for _, line := range inputLines {
		inputStream = append(inputStream, line...)
	}

	// initialize reader
	buffer := bytes.NewBuffer(inputStream)
	codec, _ := encoding.Plain(buffer)
	config := Config{
		Codec:      codec,
		Separator:  []byte("\n"),
		BufferSize: buffer.Len(),
	}
	reader, err := NewLineReader(buffer, config)
	if err != nil {
		t.Fatalf("Error initializing reader: %v", err)
	}

	// read lines
	var lines [][]byte
	for range inputLines {
		bytes, _, err := reader.Next()
		if err != nil {
			t.Fatalf("failed to read all lines from test: %v", err)
		}

		lines = append(lines, bytes)
	}

	// validate
	for i := range inputLines {
		assert.Equal(t, len(inputLines[i]), len(lines[i]))
		assert.Equal(t, inputLines[i], lines[i])
	}
}

func testReadLine(t *testing.T, line []byte) {
	testReadLines(t, [][]byte{line})
}
