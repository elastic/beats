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

package diskqueue

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSegmentsRoundTrip(t *testing.T) {
	tests := map[string]struct {
		id        segmentID
		encrypt   bool
		compress  bool
		plaintext []byte
	}{
		"No Encryption or Compression": {
			id:        0,
			encrypt:   false,
			compress:  false,
			plaintext: []byte("no encryption or compression"),
		},
		"Encryption Only": {
			id:        1,
			encrypt:   true,
			compress:  false,
			plaintext: []byte("encryption only"),
		},
		"Compression Only": {
			id:        2,
			encrypt:   false,
			compress:  true,
			plaintext: []byte("compression only"),
		},
		"Encryption and Compression": {
			id:        3,
			encrypt:   true,
			compress:  true,
			plaintext: []byte("encryption and compression"),
		},
	}
	dir := t.TempDir()
	for name, tc := range tests {
		dst := make([]byte, len(tc.plaintext))
		settings := DefaultSettings()
		settings.Path = dir
		if tc.encrypt {
			settings.EncryptionKey = []byte("keykeykeykeykeyk")
		}
		settings.UseCompression = tc.compress
		qs := &queueSegment{
			id: tc.id,
		}
		sw, err := qs.getWriter(settings)
		assert.Nil(t, err, name)

		n, err := sw.Write(tc.plaintext)
		assert.Nil(t, err, name)
		assert.Equal(t, len(tc.plaintext), n, name)

		err = sw.Close()
		assert.Nil(t, err, name)

		sr, err := qs.getReader(settings)
		assert.Nil(t, err, name)

		n, err = sr.Read(dst)
		assert.Nil(t, err, name)

		assert.Equal(t, len(dst), n, name)

		//make sure we read back what we wrote
		assert.Equal(t, tc.plaintext, dst, name)

		_, err = sr.Read(dst)
		assert.ErrorIs(t, err, io.EOF, name)

		err = sr.Close()
		assert.Nil(t, err, name)

	}
}

func TestSegmentReaderSeek(t *testing.T) {
	tests := map[string]struct {
		id         segmentID
		encrypt    bool
		compress   bool
		plaintexts [][]byte
	}{
		"No Encryption or compression": {
			id:         0,
			encrypt:    false,
			compress:   false,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
		},
		"Encryption Only": {
			id:         1,
			encrypt:    true,
			compress:   false,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
		},
		"Compression Only": {
			id:         2,
			encrypt:    false,
			compress:   true,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
		},
		"Encryption and Compression": {
			id:         3,
			encrypt:    true,
			compress:   true,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
		},
	}
	dir := t.TempDir()
	for name, tc := range tests {
		settings := DefaultSettings()
		settings.Path = dir
		if tc.encrypt {
			settings.EncryptionKey = []byte("keykeykeykeykeyk")
		}
		settings.UseCompression = tc.compress

		qs := &queueSegment{
			id: tc.id,
		}
		sw, err := qs.getWriter(settings)
		assert.Nil(t, err, name)
		for _, plaintext := range tc.plaintexts {
			n, err := sw.Write(plaintext)
			assert.Nil(t, err, name)
			assert.Equal(t, len(plaintext), n, name)
			err = sw.Sync()
			assert.Nil(t, err, name)
		}
		sw.Close()
		sr, err := qs.getReader(settings)
		assert.Nil(t, err, name)
		//seek to second data piece
		n, err := sr.Seek(segmentHeaderSize+int64(len(tc.plaintexts[0])), io.SeekStart)
		assert.Nil(t, err, name)
		assert.Equal(t, segmentHeaderSize+int64(len(tc.plaintexts[0])), n, name)
		dst := make([]byte, len(tc.plaintexts[1]))

		_, err = sr.Read(dst)
		assert.Nil(t, err, name)
		assert.Equal(t, tc.plaintexts[1], dst, name)

		sw.Close()
	}
}

func TestSegmentReaderSeekLocations(t *testing.T) {
	tests := map[string]struct {
		id         segmentID
		encrypt    bool
		compress   bool
		plaintexts [][]byte
		location   int64
	}{
		"No Encryption or Compression": {
			id:         0,
			encrypt:    false,
			compress:   false,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
			location:   -1,
		},
		"Encryption": {
			id:         1,
			encrypt:    true,
			compress:   false,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
			location:   2,
		},
		"Compression": {
			id:         1,
			encrypt:    false,
			compress:   true,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
			location:   2,
		},
		"Encryption and Compression": {
			id:         1,
			encrypt:    true,
			compress:   true,
			plaintexts: [][]byte{[]byte("abc"), []byte("defg")},
			location:   2,
		},
	}
	dir := t.TempDir()
	for name, tc := range tests {
		settings := DefaultSettings()
		settings.Path = dir
		if tc.encrypt {
			settings.EncryptionKey = []byte("keykeykeykeykeyk")
		}
		settings.UseCompression = tc.compress
		qs := &queueSegment{
			id: tc.id,
		}
		sw, err := qs.getWriter(settings)
		assert.Nil(t, err, name)
		for _, plaintext := range tc.plaintexts {
			n, err := sw.Write(plaintext)
			assert.Nil(t, err, name)
			assert.Equal(t, len(plaintext), n, name)
		}
		sw.Close()
		sr, err := qs.getReader(settings)
		assert.Nil(t, err, name)
		//seek to location
		_, err = sr.Seek(tc.location, io.SeekStart)
		assert.NotNil(t, err, name)
	}
}
