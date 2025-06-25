package filestream

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	magicBytes   = []byte(magicHeader)
	plainContent = []byte("People assume that time is a strict progression of cause to effect, but actually from a non-linear, non-subjective viewpoint, it's more like a big ball of wibbly-wobbly, timey-wimey stuff.")
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
				name:          "SeekStart when current offset > target offset (backward seek requiring reset)",
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
				name:          "SeekStart beyond EOF should error",
				buffSize:      512,
				initialRead:   0,
				seekOffset:    300, // larger than len(plainContent) = 188
				seekWhence:    io.SeekStart,
				wantOffset:    188, // actual data length
				wantReadData:  "",
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
				if tc.wantEOF {
					require.ErrorIs(t, err, io.EOF)
					assert.Equal(t, tc.wantOffset, gotOffset)
					return
				} else {
					require.NoError(t, err, "Seek(%d, %d) should not fail",
						tc.seekOffset, tc.seekWhence)
					require.Equal(t, tc.wantOffset, gotOffset, "Seek offset mismatch")
				}

				// Read after seek to verify position and data
				readBuf := make([]byte, tc.readAfterSeek)
				n, err := gsr.Read(readBuf)
				require.NoError(t, err, "Read after seek should not fail")
				assert.Equal(t, tc.readAfterSeek, n, "Read count mismatch after seek")
				assert.Equal(t, tc.wantReadData, string(readBuf), "Content mismatch after seek")
			})
		}
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
