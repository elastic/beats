// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkgutil

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/xar"
)

// cpioEntry describes a single odc ("070707") cpio archive member.
type cpioEntry struct {
	path string
	mode uint64
	body []byte
}

// writeCPIO encodes the given entries (plus the terminating TRAILER record)
// into a POSIX odc cpio archive, matching the format expected by cpio.Reader.
func writeCPIO(t *testing.T, entries []cpioEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	// In the odc format every header field is fixed-width ASCII octal: 6 chars
	// for all fields except mtime and filesize, which are 11 chars. Fields not
	// relevant to extraction (dev, ino, uid, gid, rdev) are written as zero.
	writeEntry := func(path string, mode uint64, body []byte) {
		name := append([]byte(path), 0)                  // file name, NUL terminated
		buf.WriteString("070707")                        // magic identifying the odc format
		buf.WriteString("000000")                        // dev
		buf.WriteString("000000")                        // ino
		buf.WriteString(fmt.Sprintf("%06o", mode))       // mode (file type + perm bits)
		buf.WriteString("000000")                        // uid
		buf.WriteString("000000")                        // gid
		buf.WriteString("000001")                        // nlink (1 link)
		buf.WriteString("000000")                        // rdev
		buf.WriteString(fmt.Sprintf("%011o", 0))         // mtime (epoch 0, irrelevant here)
		buf.WriteString(fmt.Sprintf("%06o", len(name)))  // namesize, includes the trailing NUL
		buf.WriteString(fmt.Sprintf("%011o", len(body))) // filesize
		buf.Write(name)
		buf.Write(body)
	}

	for _, e := range entries {
		writeEntry(e.path, e.mode, e.body)
	}
	// Trailer record terminates the archive.
	writeEntry("TRAILER!!!", 0, nil)

	return buf.Bytes()
}

// gzipPayload wraps the cpio bytes in a gzip stream and returns it as a
// xar.File, mirroring the Payload member of a real macOS .pkg.
func gzipPayload(t *testing.T, raw []byte) *xar.File {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(raw)
	require.NoError(t, err, "failed to gzip cpio payload")
	require.NoError(t, gw.Close(), "failed to close gzip writer")

	return &xar.File{Name: "Payload", Body: bytes.NewReader(buf.Bytes())}
}

// cpio mode values combine the POSIX file-type bits with permission bits.
const (
	modeRegular = 0100644 // S_IFREG (regular file) | 0644
	modeDir     = 0040755 // S_IFDIR (directory) | 0755
)

func TestExpandPayloadExtractsFiles(t *testing.T) {
	dstDir := t.TempDir()

	payload := gzipPayload(t, writeCPIO(t, []cpioEntry{
		{path: "dir", mode: modeDir},
		{path: "dir/file.txt", mode: modeRegular, body: []byte("hello")},
	}))

	err := expandPayload(payload, dstDir)
	require.NoError(t, err, "expandPayload should extract a well-formed payload")

	got, err := os.ReadFile(filepath.Join(dstDir, "dir", "file.txt"))
	require.NoError(t, err, "extracted file should exist")
	assert.Equal(t, "hello", string(got), "extracted file content should match the archive")
}

func TestExpandPayloadRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name  string
		entry cpioEntry
	}{
		{
			name:  "regular file escapes via parent dir",
			entry: cpioEntry{path: "../escape.txt", mode: modeRegular, body: []byte("pwned")},
		},
		{
			name:  "directory escapes via parent dir",
			entry: cpioEntry{path: "../../escape-dir", mode: modeDir},
		},
		{
			name:  "nested traversal escapes",
			entry: cpioEntry{path: "a/../../escape.txt", mode: modeRegular, body: []byte("pwned")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			dstDir := filepath.Join(parent, "dst")

			payload := gzipPayload(t, writeCPIO(t, []cpioEntry{tc.entry}))

			err := expandPayload(payload, dstDir)
			require.Error(t, err, "expandPayload should reject path traversal entries")
			assert.Contains(t, err.Error(), "illegal file path in pkg payload",
				"error should indicate an illegal path")

			// Nothing must be written outside the destination directory.
			escaped, statErr := os.ReadDir(parent)
			require.NoError(t, statErr, "parent dir should be readable")
			for _, e := range escaped {
				assert.Equal(t, "dst", e.Name(),
					"no entry should be written outside the destination dir")
			}
		})
	}
}
