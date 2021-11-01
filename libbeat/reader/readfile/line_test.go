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

package readfile

import (
	"bytes"
	"encoding/hex"
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
	encoding string
	strings  []string
}

var tests = []lineTestCase{
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
	runTest := func(t *testing.T, test lineTestCase) {
		codecFactory, ok := encoding.FindEncoding(test.encoding)
		if !ok {
			t.Fatalf("can not find encoding '%v'", test.encoding)
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
		reader, err := NewLineReader(ioutil.NopCloser(buffer), Config{codec, 1024, LineFeed, unlimited})
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

	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			runTest(t, test)
		})
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

		reader, err := NewLineReader(ioutil.NopCloser(buffer), Config{codec, 1024, terminator, unlimited})
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
	reader, err := NewLineReader(ioutil.NopCloser(r), Config{codec, buffer.Len(), LineFeed, unlimited})
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
	testReadLines(t, [][]byte{line}, false)
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
	reader, err := NewLineReader(ioutil.NopCloser(strings.NewReader(input)), Config{codec, bufferSize, LineFeed, lineMaxLimit})
	if err != nil {
		t.Fatal("failed to initialize reader:", err)
	}

	// Read decodec lines and test
	var idx int
	for i := 0; ; i++ {
		b, _, err := reader.Next()
		if err != nil {
			if err == io.EOF {
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

		s := string(b[:len(b)-len(nl)])
		if line != s {
			t.Fatalf("lines do not match, expected: %s got: %s", line, s)
		}
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
	reader, err := NewLineReader(in, Config{codec, bufferSize, AutoLineTerminator, 1024})
	if err != nil {
		t.Fatal("failed to initialize reader:", err)
	}

	for i := 0; i < len(lines); i++ {
		b, n, err := reader.Next()
		if err != nil {
			if err == io.EOF {
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
