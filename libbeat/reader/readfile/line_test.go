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
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/reader/readfile/encoding"
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
	{"plain", []string{"I can", "eat glass"}},
	{"iso8859-1", []string{"Filebeat is my favourite"}},
	{"iso8859-2", []string{"Filebeat je môj obľúbený"}},                          // slovak: filebeat is my favourite
	{"iso8859-3", []string{"büyükannem Filebeat kullanıyor"}},                    // turkish: my granmother uses filebeat
	{"iso8859-4", []string{"Filebeat on mõeldud kõigile"}},                       // estonian: filebeat is for everyone
	{"iso8859-5", []string{"я люблю кодировки"}},                                 // russian: i love encodings
	{"iso8859-6", []string{"أنا بحاجة إلى المزيد من الترميزات"}},                 // arabic: i need more encodings
	{"iso8859-7", []string{"όπου μπορώ να αγοράσω περισσότερες κωδικοποιήσεις"}}, // greek: where can i buy more encodings?
	{"iso8859-8", []string{"אני צריך קידוד אישי"}},                               // hebrew: i need a personal encoding
	{"iso8859-9", []string{"kodlamaları pişirebilirim"}},                         // turkish: i can cook encodings
	{"iso8859-10", []string{"koodaukset jäädyttävät nollaan"}},                   // finnish: encodings freeze below zero
	{"iso8859-13", []string{"mój pies zjada kodowanie"}},                         // polish: my dog eats encodings
	{"iso8859-14", []string{"An féidir leat cáise a ionchódú?"}},                 // irish: can you encode a cheese?
	{"iso8859-15", []string{"bedes du kode", "for min €"}},                       // danish: please encode my euro symbol
	{"iso8859-16", []string{"rossz karakterkódolást", "használsz"}},              // hungarian: you use the wrong character encoding
	{"koi8r", []string{"я люблю кодировки"}},                                     // russian: i love encodings
	{"koi8u", []string{"я люблю кодировки"}},                                     // russian: i love encodings
	{"windows1250", []string{"Filebeat je môj obľúbený"}},                        // slovak: filebeat is my favourite
	{"windows1251", []string{"я люблю кодировки"}},                               // russian: i love encodings
	{"windows1252", []string{"what is better than an encoding?", "a legacy encoding"}},
	{"windows1253", []string{"όπου μπορώ να αγοράσω", "περισσότερες κωδικοποιήσεις"}}, // greek: where can i buy more encodings?
	{"windows1254", []string{"kodlamaları", "pişirebilirim"}},                         // turkish: i can cook encodings
	{"windows1255", []string{"אני צריך קידוד אישי"}},                                  // hebrew: i need a personal encoding
	{"windows1256", []string{"أنا بحاجة إلى المزيد من الترميزات"}},                    // arabic: i need more encodings
	{"windows1257", []string{"toite", "kodeerijaid"}},                                 // estonian: feed the encoders
}

func TestReaderEncodings(t *testing.T) {
	for _, test := range tests {
		t.Logf("test codec: %v", test.encoding)

		codecFactory, ok := encoding.FindEncoding(test.encoding)
		if !ok {
			t.Errorf("can not find encoding '%v'", test.encoding)
			continue
		}

		buffer := bytes.NewBuffer(nil)
		codec, _ := codecFactory(buffer)
		nl := lineTerminatorCharacters[LineFeed]

		// write with encoding to buffer
		writer := transform.NewWriter(buffer, codec.NewEncoder())
		var expectedCount []int
		for _, line := range test.strings {
			writer.Write([]byte(line))
			writer.Write(nl)
			expectedCount = append(expectedCount, buffer.Len())
		}

		// create line reader
		reader, err := NewLineReader(buffer, Config{codec, 1024, LineFeed})
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}

		// read decodec lines from buffer
		var readLines []string
		var byteCounts []int
		current := 0
		for {
			bytes, sz, err := reader.Next()
			if sz > 0 {
				readLines = append(readLines, string(bytes[:len(bytes)-len(nl)]))
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

func TestLineTerminators(t *testing.T) {
	codecFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Errorf("can not find plain encoding")
	}

	buffer := bytes.NewBuffer(nil)
	codec, _ := codecFactory(buffer)

	for terminator, nl := range lineTerminatorCharacters {
		buffer.Reset()

		buffer.Write([]byte("this is my first line"))
		buffer.Write(nl)
		buffer.Write([]byte("this is my second line"))
		buffer.Write(nl)

		reader, err := NewLineReader(buffer, Config{codec, 1024, terminator})
		if err != nil {
			t.Errorf("failed to initialize reader: %v", err)
			continue
		}

		nrLines := 0
		for {
			line, _, err := reader.Next()
			if err != nil {
				break
			}

			assert.True(t, bytes.HasSuffix(line, nl))
			nrLines++
		}
		assert.Equal(t, nrLines, 2, "unexpected number of lines for terminator %+v", terminator)
	}
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
	reader, err := NewLineReader(buffer, Config{codec, buffer.Len(), LineFeed})
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
