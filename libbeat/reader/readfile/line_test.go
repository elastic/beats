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

package readfile

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

// Sample texts are from http://www.columbia.edu/~kermit/utf8.html
type lineTestCase struct {
	encoding       string
	strings        []string
	collectOnEOF   bool
	withEOL        bool
	lineTerminator LineTerminator
}

var tests = []lineTestCase{
	{encoding: "plain", strings: []string{"I can", "eat glass"}},
	{encoding: "latin1", strings: []string{"I kå Glas frässa", "ond des macht mr nix!"}},
	{encoding: "utf-16be", strings: []string{"Pot să mănânc sticlă", "și ea nu mă rănește."}},
	{encoding: "utf-16le", strings: []string{"काचं शक्नोम्यत्तुम् ।", "नोपहिनस्ति माम् ॥"}},
	{encoding: "big5", strings: []string{"我能吞下玻", "璃而不傷身體。"}},
	{encoding: "gb18030", strings: []string{"我能吞下玻璃", "而不傷身。體"}},
	{encoding: "euc-kr", strings: []string{" 나는 유리를 먹을 수 있어요.", " 그래도 아프지 않아요"}},
	{encoding: "euc-jp", strings: []string{"私はガラスを食べられます。", "それは私を傷つけません。"}},
	{encoding: "plain", strings: []string{"I can", "eat glass"}},
	{encoding: "iso8859-1", strings: []string{"Filebeat is my favourite"}},
	{encoding: "iso8859-2", strings: []string{"Filebeat je môj obľúbený"}},                          // slovak: filebeat is my favourite
	{encoding: "iso8859-3", strings: []string{"büyükannem Filebeat kullanıyor"}},                    // turkish: my granmother uses filebeat
	{encoding: "iso8859-4", strings: []string{"Filebeat on mõeldud kõigile"}},                       // estonian: filebeat is for everyone
	{encoding: "iso8859-5", strings: []string{"я люблю кодировки"}},                                 // russian: i love encodings
	{encoding: "iso8859-6", strings: []string{"أنا بحاجة إلى المزيد من الترميزات"}},                 // arabic: i need more encodings
	{encoding: "iso8859-7", strings: []string{"όπου μπορώ να αγοράσω περισσότερες κωδικοποιήσεις"}}, // greek: where can i buy more encodings?
	{encoding: "iso8859-8", strings: []string{"אני צריך קידוד אישי"}},                               // hebrew: i need a personal encoding
	{encoding: "iso8859-9", strings: []string{"kodlamaları pişirebilirim"}},                         // turkish: i can cook encodings
	{encoding: "iso8859-10", strings: []string{"koodaukset jäädyttävät nollaan"}},                   // finnish: encodings freeze below zero
	{encoding: "iso8859-13", strings: []string{"mój pies zjada kodowanie"}},                         // polish: my dog eats encodings
	{encoding: "iso8859-14", strings: []string{"An féidir leat cáise a ionchódú?"}},                 // irish: can you encode a cheese?
	{encoding: "iso8859-15", strings: []string{"bedes du kode", "for min €"}},                       // danish: please encode my euro symbol
	{encoding: "iso8859-16", strings: []string{"rossz karakterkódolást", "használsz"}},              // hungarian: you use the wrong character encoding
	{encoding: "koi8r", strings: []string{"я люблю кодировки"}},                                     // russian: i love encodings
	{encoding: "koi8u", strings: []string{"я люблю кодировки"}},                                     // russian: i love encodings
	{encoding: "windows1250", strings: []string{"Filebeat je môj obľúbený"}},                        // slovak: filebeat is my favourite
	{encoding: "windows1251", strings: []string{"я люблю кодировки"}},                               // russian: i love encodings
	{encoding: "windows1252", strings: []string{"what is better than an encoding?", "a legacy encoding"}},
	{encoding: "windows1253", strings: []string{"όπου μπορώ να αγοράσω", "περισσότερες κωδικοποιήσεις"}}, // greek: where can i buy more encodings?
	{encoding: "windows1254", strings: []string{"kodlamaları", "pişirebilirim"}},                         // turkish: i can cook encodings
	{encoding: "windows1255", strings: []string{"אני צריך קידוד אישי"}},                                  // hebrew: i need a personal encoding
	{encoding: "windows1256", strings: []string{"أنا بحاجة إلى المزيد من الترميزات"}},                    // arabic: i need more encodings
	{encoding: "windows1257", strings: []string{"toite", "kodeerijaid"}},                                 // estonian: feed the encoders
}

func TestReaderEncodings(t *testing.T) {
	runTest := func(t *testing.T, test lineTestCase) {
		codecFactory, ok := encoding.FindEncoding(test.encoding)
		if !ok {
			t.Fatalf("can not find encoding '%v'", test.encoding)
		}

		buffer := bytes.NewBuffer(nil)
		codec, _ := codecFactory(buffer)
		nl := lineTerminatorCharacters[test.lineTerminator]

		// write with encoding to buffer
		writer := transform.NewWriter(buffer, codec.NewEncoder())
		var expectedCount []int
		for i, line := range test.strings {
			_, _ = writer.Write([]byte(line))
			if !test.collectOnEOF || i < len(test.strings)-1 || test.withEOL {
				_, _ = writer.Write(nl)
			}
			expectedCount = append(expectedCount, buffer.Len())
		}

		// create line reader
		reader, err := NewLineReader(ioutil.NopCloser(buffer), Config{codec, 1024, test.lineTerminator, unlimited, test.collectOnEOF})
		if err != nil {
			t.Fatal("failed to initialize reader:", err)
		}

		// read decodec lines from buffer
		var readLines []string
		var byteCounts []int
		current := 0
		for {
			bytes, sz, err := reader.Next()
			if sz > 0 {
				offset := len(bytes)
				if offset > 0 && (!test.collectOnEOF || !errors.Is(err, io.EOF) || test.withEOL) {
					offset -= len(nl)
				}
				readLines = append(readLines, string(bytes[:offset]))
			}

			current += sz
			byteCounts = append(byteCounts, current)

			if err != nil {
				break
			}
		}

		// validate lines and byte offsets
		if len(test.strings) != len(readLines) {
			t.Fatalf("number of lines mismatch (expected=%v actual=%v)",
				len(test.strings), len(readLines))
		}
		for i := range test.strings {
			expected := test.strings[i]
			actual := readLines[i]
			assert.Equal(t, expected, actual)
			assert.Equal(t, expectedCount[i], byteCounts[i])
		}
	}

	invalidLineTerminatorForEncoding := map[string][]LineTerminator{
		"latin1":      []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"big5":        []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"euc-kr":      []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"euc-jp":      []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-1":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-2":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-3":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-4":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-5":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-6":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-7":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-8":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-9":   []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-10":  []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-13":  []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-14":  []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-15":  []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"iso8859-16":  []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"koi8r":       []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"koi8u":       []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1250": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1251": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1252": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1253": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1254": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1255": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1256": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"windows1257": []LineTerminator{LineSeparator, NextLine, ParagraphSeparator},
		"utf-16be":    []LineTerminator{NextLine}, // test fails: buf ends with uint8{189} instead of uint8{133}
		"gb18030":     []LineTerminator{NextLine}, // test fails: buf ends with uint8{189} instead of uint8{133}
		"utf-16le":    []LineTerminator{NextLine}, // test fails: buf ends with uint8{189} instead of uint8{133}
	}
	for _, test := range tests {
		for _, collectOnEOF := range []bool{false, true} {
			for _, withEOL := range []bool{false, true} {
				for lineTerminatorName, lineTerminator := range lineTerminators {
					lineTerminatorIsInvalid := false
					if invalidLineTerminatorForEncoding, ok := invalidLineTerminatorForEncoding[test.encoding]; ok {
						for _, invalidLineTerminator := range invalidLineTerminatorForEncoding {
							if invalidLineTerminator == lineTerminator {
								lineTerminatorIsInvalid = true
								break
							}
						}
					}
					if lineTerminatorIsInvalid {
						continue
					}

					test.withEOL = withEOL
					test.collectOnEOF = collectOnEOF
					test.lineTerminator = lineTerminator
					t.Run(fmt.Sprintf("encoding: %s, collect on EOF: %t, with EOL: %t, line terminator: %s", test.encoding, test.collectOnEOF, test.withEOL, lineTerminatorName), func(t *testing.T) {
						runTest(t, test)
					})
				}
			}
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

		reader, err := NewLineReader(ioutil.NopCloser(buffer), Config{codec, 1024, terminator, unlimited, false})
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

	testReadLines(t, lines, false)
}

func testReadLines(t *testing.T, inputLines [][]byte, eofOnLastRead bool) {
	var inputStream []byte
	for _, line := range inputLines {
		inputStream = append(inputStream, line...)
	}

	// initialize reader
	buffer := bytes.NewBuffer(inputStream)

	var r io.Reader = buffer
	if eofOnLastRead {
		r = &eofWithNonZeroNumberOfBytesReader{buf: buffer}
	}

	codec, _ := encoding.Plain(r)
	reader, err := NewLineReader(ioutil.NopCloser(r), Config{codec, buffer.Len(), LineFeed, unlimited, false})
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

func randomInt(r *rand.Rand, min, max int) int {
	return r.Intn(max+1-min) + min
}

func randomBool(r *rand.Rand) bool {
	n := randomInt(r, 0, 1)
	return n != 0
}

func randomBytes(r *rand.Rand, sz int) ([]byte, error) {
	bytes := make([]byte, sz)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func randomString(r *rand.Rand, sz int) (string, error) {
	if sz == 0 {
		return "", nil
	}

	var bytes []byte
	var err error
	if bytes, err = randomBytes(r, sz/2+sz%2); err != nil {
		return "", err
	}
	s := hex.EncodeToString(bytes)
	return s[:sz], nil
}

func setupTestMaxBytesLimit(lineMaxLimit, lineLen int, nl []byte) (lines []string, data string, err error) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	lineCount := randomInt(rnd, 11, 142)
	lines = make([]string, lineCount)

	var b strings.Builder

	for i := 0; i < lineCount; i++ {
		var sz int
		// Non-empty line
		if randomBool(rnd) {
			// Boundary to the lineMaxLimit
			if randomBool(rnd) {
				sz = randomInt(rnd, lineMaxLimit-1, lineMaxLimit+1)
			} else {
				sz = randomInt(rnd, 0, lineLen)
			}
		} else {
			// Randomly empty or one characters lines(another possibly boundary conditions)
			sz = randomInt(rnd, 0, 1)
		}

		s, err := randomString(rnd, sz)
		if err != nil {
			return nil, "", err
		}

		lines[i] = s
		if len(s) > 0 {
			b.WriteString(s)
		}
		b.Write(nl)
	}
	return lines, b.String(), nil
}

func TestMaxBytesLimit(t *testing.T) {
	const (
		enc           = "plain"
		numberOfLines = 102
		bufferSize    = 1024
		lineMaxLimit  = 3012
		lineLen       = 5720 // exceeds lineMaxLimit
	)

	codecFactory, ok := encoding.FindEncoding(enc)
	if !ok {
		t.Fatalf("can not find encoding '%v'", enc)
	}

	buffer := bytes.NewBuffer(nil)
	codec, _ := codecFactory(buffer)
	nl := lineTerminatorCharacters[LineFeed]

	// Generate random lines lengths including empty lines
	lines, input, err := setupTestMaxBytesLimit(lineMaxLimit, lineLen, nl)
	if err != nil {
		t.Fatal("failed to generate random input:", err)
	}

	// Create line reader
	reader, err := NewLineReader(ioutil.NopCloser(strings.NewReader(input)), Config{codec, bufferSize, LineFeed, lineMaxLimit, false})
	if err != nil {
		t.Fatal("failed to initialize reader:", err)
	}

	// Read decodec lines and test
	var (
		idx     int
		readLen int
	)

	for i := 0; ; i++ {
		b, n, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				readLen += n
				break
			} else {
				t.Fatal("unexpected error:", err)
			}
		}

		// Find the next expected line from the original test array
		var line string
		for ; idx < len(lines); idx++ {
			// Expected to be dropped
			if len(lines[idx]) > lineMaxLimit {
				continue
			}
			line = lines[idx]
			idx++
			break
		}

		readLen += n
		s := string(b[:len(b)-len(nl)])
		if line != s {
			t.Fatalf("lines do not match, expected: %s got: %s", line, s)
		}
	}

	if len(input) != readLen {
		t.Fatalf("the bytes read are not equal to the bytes input, expected: %d got: %d", len(input), readLen)
	}
}

// test_exceed_buffer from test_harvester.py
func TestBufferSize(t *testing.T) {
	lines := []string{
		"first line is too long\n",
		"second line is too long\n",
		"third line too long\n",
		"OK\n",
	}

	codecFactory, _ := encoding.FindEncoding("")
	codec, _ := codecFactory(bytes.NewBuffer(nil))
	bufferSize := 10

	in := ioutil.NopCloser(strings.NewReader(strings.Join(lines, "")))
	reader, err := NewLineReader(in, Config{codec, bufferSize, AutoLineTerminator, 1024, false})
	if err != nil {
		t.Fatal("failed to initialize reader:", err)
	}

	for i := 0; i < len(lines); i++ {
		b, n, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				t.Fatal("unexpected error:", err)
			}
		}

		require.Equal(t, n, len(lines[i]))
		require.Equal(t, string(b[:n]), lines[i])
	}
}

// eofWithNonZeroNumberOfBytesReader is an io.Reader implementation that at the
// end of the stream returns a non-zero number of bytes with io.EOF. This is
// allowed under the io.Reader interface contract and must be handled by the
// line reader.
type eofWithNonZeroNumberOfBytesReader struct {
	buf *bytes.Buffer
}

func (r *eofWithNonZeroNumberOfBytesReader) Read(d []byte) (int, error) {
	n, err := r.buf.Read(d)
	if err != nil {
		return n, err
	}

	// As per the io.Reader contract:
	//   "a Reader returning a non-zero number of bytes at the end of the input
	//   stream may return either err == EOF or err == nil."
	if r.buf.Len() == 0 {
		return n, io.EOF
	}
	return n, nil
}

// Verify handling of the io.Reader returning n > 0 with io.EOF.
func TestReadWithNonZeroNumberOfBytesAndEOF(t *testing.T) {
	testReadLines(t, [][]byte{[]byte("Hello world!\n")}, true)
}
