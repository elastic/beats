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

package filestream

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
)

var (
	magicBytes   = []byte(magicHeader)
	plainContent = []byte(
		"People assume that time is a strict progression of cause to effect, " +
			"but actually from a non-linear, non-subjective viewpoint, it's " +
			"more like a big ball of wibbly-wobbly, timey-wimey stuff.")
)

var _ File = (*plainFile)(nil)
var _ File = (*gzipSeekerReader)(nil)

func TestPlainFile(t *testing.T) {
	testContent := []byte("hello world")
	osFile := createAndOpenFile(t, testContent)

	pf := newPlainFile(osFile)

	t.Run("IsGZIP returns false", func(t *testing.T) {
		assert.False(t, pf.IsGZIP())
	})

	t.Run("OSFile returns underlying os.File", func(t *testing.T) {
		assert.Exactly(t, osFile, pf.OSFile())
	})
}

func TestGzipSeekerReader(t *testing.T) {
	t.Run("newGzipSeekerReader success", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err)
		require.NotNil(t, gsr)
	})

	t.Run("newGzipSeekerReader error on non-gzip file", func(t *testing.T) {
		osFile := createAndOpenFile(t, []byte("not gzip content"))

		gsr, err := newGzipSeekerReader(osFile, 1024)
		assert.Error(t, err)
		assert.Nil(t, gsr)
		assert.Contains(t, err.Error(), "could not create gzip reader")
		assert.Contains(t, err.Error(), gzip.ErrHeader.Error())
	})
	t.Run("IsGZIP returns true", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err)

		assert.True(t, gsr.IsGZIP())
	})

	t.Run("OSFile returns underlying os.File", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err)

		assert.Exactly(t, osFile, gsr.OSFile())
	})

	t.Run("Stat proxies to underlying file", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err)

		gsrFi, err := gsr.Stat()
		require.NoError(t, err)

		osFi, err := osFile.Stat()
		require.NoError(t, err)

		assert.Equal(t, osFi.Name(), gsrFi.Name())
		assert.Equal(t, osFi.Size(), gsrFi.Size())
	})

	t.Run("Name proxies to underlying file", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err)

		assert.Equal(t, osFile.Name(), gsr.Name())
	})

	t.Run("Read reads decompressed content", func(t *testing.T) {
		osFile := createAndOpenFile(t, newGzippedDataSource(t))
		gsr, err := newGzipSeekerReader(osFile, 1024)
		require.NoError(t, err, "could not create gzip seeker reader")

		readBuf := make([]byte, len(plainContent))
		n, err := gsr.Read(readBuf)
		if !errors.Is(err, io.EOF) {
			require.NoError(t, err)
		}

		assert.Equal(t, len(plainContent), n)
		assert.Equal(t, string(plainContent), string(readBuf))
	})

	t.Run("Read all data on integrity validation (CRC/size) error", func(t *testing.T) {
		var content []byte
		nl := []byte("\n")
		buffSize := len(plainContent) + len(nl)

		content = append(content, plainContent...)
		content = append(content, nl...)
		content = append(content, plainContent...)
		content = append(content, nl...)
		corrupted := gziptest.Compress(
			t,
			content,
			gziptest.CorruptCRC)
		osFile := createAndOpenFile(t, corrupted)
		gsr, err := newGzipSeekerReader(osFile, buffSize)
		require.NoError(t, err, "could not create gzip seeker reader")

		buff := make([]byte, buffSize)

		n, err := gsr.Read(buff)
		assert.Equal(t, buffSize, n, "read data should match line size")
		assert.Equal(t, buff, append(plainContent, nl...),
			"1st read should read the whole first line")
		assert.NoError(t, err, "1st read should not return error")

		n, err = gsr.Read(buff)
		assert.Equal(t, buffSize, n, "read data should match line size")
		assert.Equal(t, buff, append(plainContent, nl...),
			"2nd read should read the whole second line")
		assert.ErrorIs(t, err, gzip.ErrChecksum, "2nd read: unexpected error")

		n, err = gsr.Read(buff)
		assert.Equal(t, 0, n, "3rd read should not read any data")
		assert.ErrorIs(t, err, gzip.ErrChecksum, "3rd read: unexpected error")
	})

	t.Run("Seek", func(t *testing.T) {
		tests := []struct {
			name          string
			buffSize      int
			initialRead   int // bytes to read before seek
			seekOffset    int64
			seekWhence    int
			wantOffset    int64
			wantReadData  string // expected data after seek+read
			readAfterSeek int    // bytes to read after seek for verification
			wantEOF       bool   // whether Seek is expected to EOF error
			wantErr       string // want see to return an error
		}{
			{
				name:          "SeekStart to offset 0 after reading some data (reset case)",
				buffSize:      512,
				initialRead:   10,
				seekOffset:    0,
				seekWhence:    io.SeekStart,
				wantOffset:    0,
				wantReadData:  string(plainContent[:10]),
				readAfterSeek: 10,
			},
			{
				name:          "SeekStart when current offset > target offset (seek requiring reset)",
				buffSize:      512,
				initialRead:   50,
				seekOffset:    20,
				seekWhence:    io.SeekStart,
				wantOffset:    20,
				wantReadData:  string(plainContent[20:35]),
				readAfterSeek: 15,
			},
			{
				name:          "SeekCurrent with offset 0 (should stay at current position)",
				buffSize:      512,
				initialRead:   30,
				seekOffset:    0,
				seekWhence:    io.SeekCurrent,
				wantOffset:    30,
				wantReadData:  string(plainContent[30:40]),
				readAfterSeek: 10,
			},
			{
				name:          "SeekCurrent with positive offset moving forward",
				buffSize:      512,
				initialRead:   20,
				seekOffset:    15,
				seekWhence:    io.SeekCurrent,
				wantOffset:    35,
				wantReadData:  string(plainContent[35:50]),
				readAfterSeek: 15,
			},
			{
				name:          "SeekCurrent with offset that makes final position < current (backward seek requiring reset)",
				buffSize:      512,
				initialRead:   50,
				seekOffset:    -30,
				seekWhence:    io.SeekCurrent,
				wantOffset:    20,
				wantReadData:  string(plainContent[20:30]),
				readAfterSeek: 10,
			},
			{
				name:          "offset < reader buffer size",
				buffSize:      32,
				initialRead:   0,
				seekOffset:    16, // 32 / 2
				seekWhence:    io.SeekStart,
				wantOffset:    16,
				wantReadData:  string(plainContent[16:32]),
				readAfterSeek: 16,
			},
			{
				name:          "offset == n * reader buffer size",
				buffSize:      32,
				initialRead:   0,
				seekOffset:    64, // 32 * 2
				seekWhence:    io.SeekStart,
				wantOffset:    64,
				wantReadData:  string(plainContent[64:80]),
				readAfterSeek: 16,
			},
			{
				name:          "offset > n * reader buffer size + 1",
				buffSize:      32,
				initialRead:   0,
				seekOffset:    65, // 32 * 2 + 1
				seekWhence:    io.SeekStart,
				wantOffset:    65,
				wantReadData:  string(plainContent[65:81]),
				readAfterSeek: 16,
			},
			{
				name:          "SeekStart beyond EOF does not error",
				buffSize:      512,
				initialRead:   0,
				seekOffset:    300, // larger than len(plainContent) = 188
				seekWhence:    io.SeekStart,
				wantOffset:    300, // mimics os.File.Seek
				wantReadData:  "",  // No data to read as data < offset
				readAfterSeek: 0,
				wantEOF:       true,
			},
			{
				name:          "final offset is negative",
				buffSize:      512,
				initialRead:   0,
				seekOffset:    -200,
				seekWhence:    io.SeekStart,
				wantOffset:    0,
				wantReadData:  "",
				readAfterSeek: 0,
				wantErr:       "final offset must be non-negative",
			},
			{
				name:          "SeekCurrent beyond EOF does not error",
				buffSize:      512,
				initialRead:   42,
				seekOffset:    300, // larger than len(plainContent) = 188
				seekWhence:    io.SeekCurrent,
				wantOffset:    342, // mimics os.File.Seek
				wantReadData:  "",  // No data to read as data < offset
				readAfterSeek: 0,
				wantEOF:       true,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				osFile := createAndOpenFile(t, newGzippedDataSource(t))
				defer osFile.Close()

				gsr, err := newGzipSeekerReader(osFile, tc.buffSize)
				require.NoError(t, err)
				require.NotNil(t, gsr)

				// set up initial position
				if tc.initialRead > 0 {
					initialBuf := make([]byte, tc.initialRead)
					n, err := gsr.Read(initialBuf)
					require.NoError(t, err)
					require.Equal(t, tc.initialRead, n)
					require.Equal(t, string(plainContent[:tc.initialRead]), string(initialBuf))
				}

				// Seek
				gotOffset, err := gsr.Seek(tc.seekOffset, tc.seekWhence)
				if tc.wantErr != "" {
					assert.ErrorContains(t, err, tc.wantErr)
				} else {
					require.NoError(t, err, "Seek(%d, %d) should not fail",
						tc.seekOffset, tc.seekWhence)
				}
				assert.Equal(t, tc.wantOffset, gotOffset, "Seek offset mismatch")

				// Read after seek to verify position and data
				readBuf := make([]byte, tc.readAfterSeek)
				n, err := gsr.Read(readBuf)
				if tc.wantEOF {
					assert.ErrorIs(t, err, io.EOF, "unexpected error")
				} else {
					assert.NoError(t, err, "Read after seek should not fail")
				}
				assert.Equal(t, tc.readAfterSeek, n, "Read count mismatch after seek")
				assert.Equal(t, tc.wantReadData, string(readBuf), "Content mismatch after seek")
			})
		}
	})
}

// TestFileImplementations_SeekAtEOF ensures that both plain and gzip File
// implementations behave consistently when seeking to or beyond EOF.
func TestFileImplementations_SeekAtEOF(t *testing.T) {
	tempDir := t.TempDir()

	// Setup plain file
	plainFilename := filepath.Join(tempDir, "plain.txt")
	err := os.WriteFile(plainFilename, plainContent, 0644)
	require.NoError(t, err, "could not write plain file")

	// Setup gzip file
	gzipFilename := filepath.Join(tempDir, "plain.txt.gz")
	gzipFile, err := os.Create(gzipFilename)
	require.NoError(t, err, "could not create gzip file")
	gzWriter := gzip.NewWriter(gzipFile)
	_, err = gzWriter.Write(plainContent)
	require.NoError(t, err)
	require.NoError(t, gzWriter.Close())
	require.NoError(t, gzipFile.Close())

	contentLen := int64(len(plainContent))

	// buffer size chosen to hit all code dealing with advancing offset on
	// gzipSeekerReader.
	readBuffSize := 64
	t.Run("seek to exactly the end of the file", func(t *testing.T) {
		plainOSFile, err := os.Open(plainFilename)
		require.NoError(t, err)
		defer plainOSFile.Close()
		plainF := newPlainFile(plainOSFile)

		gzipOSFile, err := os.Open(gzipFilename)
		require.NoError(t, err)
		defer gzipOSFile.Close()
		gzipF, err := newGzipSeekerReader(gzipOSFile, readBuffSize)
		require.NoError(t, err)

		// Seek to EOF
		plainOffset, plainErr := plainF.Seek(contentLen, io.SeekStart)
		gzipOffset, gzipErr := gzipF.Seek(contentLen, io.SeekStart)

		assert.Equal(t, plainOffset, gzipOffset,
			"offsets should be equal when seeking to EOF")
		assert.NoError(t, plainErr,
			"no error expected when seeking to EOF")
		assert.NoError(t, gzipErr,
			"no error expected when when seeking to EOF")
		assert.Equal(t, contentLen, plainOffset, "offset should be at EOF")

		// Reading at EOF should return n=0, err=io.EOF
		p := make([]byte, 1)
		plainN, plainReadErr := plainF.Read(p)
		gzipN, gzipReadErr := gzipF.Read(p)

		assert.Equal(t, plainN, gzipN, "bytes read should be equal at EOF")
		assert.Equal(t, 0, plainN, "read should return 0 bytes at EOF")
		assert.ErrorIs(t, plainReadErr, io.EOF,
			"error should EOF")
		assert.ErrorIs(t, gzipReadErr, io.EOF,
			"error should EOF")
	})

	t.Run("seek past the end of the file", func(t *testing.T) {
		plainOSFile, err := os.Open(plainFilename)
		require.NoError(t, err)
		defer plainOSFile.Close()
		plainF := newPlainFile(plainOSFile)

		gzipOSFile, err := os.Open(gzipFilename)
		require.NoError(t, err)
		defer gzipOSFile.Close()
		gzipF, err := newGzipSeekerReader(gzipOSFile, readBuffSize)
		require.NoError(t, err)

		seekTo := contentLen + 42
		plainOffset, plainErr := plainF.Seek(seekTo, io.SeekStart)
		gzipOffset, gzipErr := gzipF.Seek(seekTo, io.SeekStart)

		assert.Equal(t, plainOffset, gzipOffset, "offsets should be equal when seeking past EOF")
		assert.Equal(t, seekTo, plainOffset, "offset should be past EOF")
		assert.NoError(t, plainErr, "no error expected when seeking past EOF")
		assert.NoError(t, gzipErr, "no error expected when seeking past EOF")

		// Reading past EOF should return n=0, err=io.EOF
		p := make([]byte, 1)
		plainN, plainReadErr := plainF.Read(p)
		gzipN, gzipReadErr := gzipF.Read(p)

		assert.Equal(t, plainN, gzipN,
			"bytes read should be equal when reading past EOF")
		assert.Equal(t, 0, plainN, "read should return 0 bytes past EOF")
		assert.ErrorIs(t, plainReadErr, io.EOF,
			"error should be EOF when reading after EOF")
		assert.ErrorIs(t, gzipReadErr, io.EOF,
			"error should be EOF when reading after EOF")
	})
}

func TestIsGZIP(t *testing.T) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte("hello gzip"))
	require.NoError(t, err, "Failed to write gzip data")
	err = gzWriter.Close()
	require.NoError(t, err, "Failed to close gzip writer")
	validGzipContent := buf.Bytes()

	var emptyContent []byte
	shortContent := magicBytes[:1]
	invalidHeaderContent := []byte{'N', 'G', 'Z', 'I', 'P'} // Not GZIP

	testCases := []struct {
		name             string
		fileContent      []byte
		initialSeek      int64
		wantIsGZIP       bool
		wantErrStr       string
		wantOffset       int64
		checkSpecificErr func(err error) bool
	}{
		{
			name:        "valid gzip file",
			fileContent: validGzipContent,
			initialSeek: 0,
			wantIsGZIP:  true,
			wantOffset:  0,
		},
		{
			name:        "valid gzip file with initial offset",
			fileContent: validGzipContent,
			initialSeek: 1,
			wantIsGZIP:  true,
			wantOffset:  1, // Offset should be restored
		},
		{
			name:        "not a gzip file - invalid header",
			fileContent: invalidHeaderContent,
			initialSeek: 0,
			wantIsGZIP:  false,
			wantOffset:  0,
		},
		{
			name:        "empty file",
			fileContent: emptyContent,
			initialSeek: 0,
			wantIsGZIP:  false, // IsGZIP handles EOF as "not gzip"
			wantOffset:  0,
		},
		{
			name:        "file shorter than magic header",
			fileContent: shortContent,
			initialSeek: 0,
			wantIsGZIP:  false, // IsGZIP handles EOF as "not gzip"
			wantOffset:  0,
		},
		{
			name:        "file with only magic header",
			fileContent: magicBytes,
			initialSeek: 0,
			wantIsGZIP:  true,
			wantOffset:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := createAndOpenFile(t, tc.fileContent)
			// File 'f' is now closed by t.Cleanup in createAndOpenFile

			originalFileOffset := tc.initialSeek
			if tc.initialSeek > 0 {
				_, err := f.Seek(tc.initialSeek, io.SeekStart)
				if err != nil {
					require.NoError(t, err, "Failed to set initial seek")
				}
			} else {
				// Get current offset if initialSeek is 0
				offset, err := f.Seek(0, io.SeekCurrent)
				require.NoError(t, err, "Failed to get initial offset")
				originalFileOffset = offset
			}

			isGzip, err := IsGZIP(f)

			if tc.wantErrStr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErrStr)
			} else if tc.checkSpecificErr != nil {
				require.Error(t, err) // Expect an error if a specific error check is defined
				require.True(t, tc.checkSpecificErr(err), "Specific error check failed for error: %v", err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.wantIsGZIP, isGzip)

			// Check if offset is restored
			currentOffset, seekErr := f.Seek(0, io.SeekCurrent)
			if tc.name != "seek error on initial seek" && tc.name != "readat error on closed file" {
				// Only require no error if we don't expect the file to be closed
				require.NoError(t, seekErr, "Failed to get current offset after IsGZIP")
				require.Equal(t, originalFileOffset, currentOffset, "File offset mismatch")
			} else if seekErr == nil {
				// If we expected a seek error (closed file) but didn't get one, that's also a problem.
				// However, if the file *wasn't* closed and seek succeeded, still check offset.
				// This branch implies seekErr == nil in a case where it might have been expected.
				// For simplicity, if seekErr is nil, always check offset.
				require.Equal(t, originalFileOffset, currentOffset, "File offset mismatch in potentially erroring test case")
			}
		})
	}

	t.Run("seek error on offset reset", func(t *testing.T) {
		f := createAndOpenFile(t, validGzipContent)
		f.Close() // Close the file to cause Seek to fail

		isGzip, err := IsGZIP(f)
		require.Error(t, err, "Expected an error when initial Seek fails")

		isClosedErr := errors.Is(err, os.ErrClosed) || strings.Contains(err.Error(),
			"file already closed")
		assert.True(t, isClosedErr,
			"Expected os.ErrClosed or 'file already closed', got: %v", err)
		assert.False(t, isGzip,
			"Expected IsGZIP to be false if file cannot be opened")
	})

	t.Run("readAt error non-EOF", func(t *testing.T) {
		tempDir := t.TempDir()
		f, err := os.Open(tempDir) // Open the directory as a file
		require.NoError(t, err, "Failed to open directory as file")
		t.Cleanup(func() {
			f.Close()
		})

		isGzip, err := IsGZIP(f)
		wantErrMsg := "GZIP: failed to read magic bytes:"

		assert.ErrorContains(t, err, wantErrMsg)
		assert.False(t, isGzip,
			"want IsGZIP to be false when ReadAt fails, got true")
	})
}

func createAndOpenFile(t *testing.T, content []byte) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "testfile-*.dat")
	require.NoError(t, err, "Failed to create temp file")

	if len(content) > 0 {
		_, err = tmpFile.Write(content)
		require.NoError(t, err, "Failed to write to temp file")
	}
	_, err = tmpFile.Seek(0, io.SeekStart)
	require.NoError(t, err, "Failed to seek to start of temp file")

	t.Cleanup(func() {
		tmpFile.Close()
	})

	return tmpFile
}

// newGzippedDataSource creates a gzipped version of plainContent.
// It's a helper function for tests that require gzipped data.
func newGzippedDataSource(t *testing.T) []byte {
	t.Helper()
	var tempBuffer bytes.Buffer
	gw := gzip.NewWriter(&tempBuffer)
	_, err := gw.Write(plainContent)
	require.NoError(t, err, "failed to write plain content to gzip writer")
	err = gw.Close()
	require.NoError(t, err, "failed to close gzip writer")
	return tempBuffer.Bytes()
}
