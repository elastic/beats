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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemasRoundTrip(t *testing.T) {
	tests := map[string]struct {
		id            segmentID
		schemaVersion uint32
		plaintext     []byte
	}{
		"version 0": {
			id:            0,
			schemaVersion: uint32(0),
			plaintext:     []byte("abc"),
		},
		"version 1": {
			id:            1,
			schemaVersion: uint32(1),
			plaintext:     []byte("abc"),
		},
		"version 2": {
			id:            2,
			schemaVersion: uint32(2),
			plaintext:     []byte("abc"),
		},
	}
	dir, err := os.MkdirTemp("", t.Name())
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	for name, tc := range tests {
		dst := make([]byte, len(tc.plaintext))
		settings := DefaultSettings()
		settings.Path = dir
		settings.SchemaVersion = tc.schemaVersion
		settings.EncryptionKey = []byte("keykeykeykeykeyk")
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

		err = sr.Close()
		assert.Nil(t, err, name)
		assert.Equal(t, len(dst), n, name)

		//make sure we read back what we wrote
		assert.Equal(t, tc.plaintext, dst, name)

	}
}

func TestSeek(t *testing.T) {
	tests := map[string]struct {
		id            segmentID
		schemaVersion uint32
		headerSize    int64
		plaintexts    [][]byte
	}{
		"version 0": {
			id:            0,
			schemaVersion: uint32(0),
			headerSize:    segmentHeaderSizeV0,
			plaintexts:    [][]byte{[]byte("abc"), []byte("defg")},
		},
		"version 1": {
			id:            1,
			schemaVersion: uint32(1),
			headerSize:    segmentHeaderSizeV1,
			plaintexts:    [][]byte{[]byte("abc"), []byte("defg")},
		},
		"version 2": {
			id:            2,
			schemaVersion: uint32(2),
			headerSize:    segmentHeaderSizeV2,
			plaintexts:    [][]byte{[]byte("abc"), []byte("defg")},
		},
	}
	dir, err := os.MkdirTemp("", t.Name())
	assert.Nil(t, err)
	//	defer os.RemoveAll(dir)
	for name, tc := range tests {
		settings := DefaultSettings()
		settings.Path = dir
		settings.SchemaVersion = tc.schemaVersion
		settings.EncryptionKey = []byte("keykeykeykeykeyk")
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
		n, err := sr.Seek(tc.headerSize+int64(len(tc.plaintexts[0])), io.SeekStart)
		assert.Nil(t, err, name)
		assert.Equal(t, tc.headerSize+int64(len(tc.plaintexts[0])), n, name)
		dst := make([]byte, len(tc.plaintexts[1]))

		_, err = sr.Read(dst)
		assert.Nil(t, err, name)
		assert.Equal(t, tc.plaintexts[1], dst, name)

		sw.Close()
	}
}
