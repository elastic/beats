// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bundle

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func newFixedResource(name string, length int64) *MemoryFile {
	paddingData, err := common.RandomBytes(int(length))
	if err != nil {
		// fatal, This mean something is wrong with the random number generator.
		panic(err)
	}
	return &MemoryFile{Path: name, Raw: paddingData, FileMode: 0755}
}

func TestZipBundle(t *testing.T) {
	t.Run("with limits", testWithLimits)
	t.Run("with no limits", testArtifact(-1, -1))
}

func testWithLimits(t *testing.T) {
	t.Run("uncompressed size is over limit", func(t *testing.T) {
		limit := int64(50)
		bundle := NewZipWithLimits(limit, -1, newFixedResource("ok.yml", limit+1))
		_, err := bundle.Bytes()
		assert.Error(t, err)
	})

	t.Run("compressed size is over limit", func(t *testing.T) {
		limit := int64(10)
		bundle := NewZipWithLimits(-1, limit, newFixedResource("ok.yml", 2*limit))
		_, err := bundle.Bytes()
		assert.Error(t, err)
	})

	t.Run("zip artifact is under limit and valid", testArtifact(1000, 1000))
}

func testArtifact(maxSizeUncompressed, maxSizeCompressed int64) func(t *testing.T) {
	return func(t *testing.T) {
		m := map[string]*MemoryFile{
			"f1.txt": newFixedResource("f1.txt", 65),
			"f2.txt": newFixedResource("f2.txt", 100),
		}

		resources := make([]Resource, len(m))
		var idx int
		for _, r := range m {
			resources[idx] = r
			idx++
		}

		bundle := NewZipWithLimits(maxSizeUncompressed, maxSizeCompressed, resources...)
		b, err := bundle.Bytes()
		if !assert.NoError(t, err) {
			return
		}

		zip, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Equal(t, 2, len(zip.File)) {
			return
		}

		for _, file := range zip.File {
			r, ok := m[file.Name]
			if !assert.True(t, ok) {
				t.Fatal("unknown file present in the zip")
			}

			reader, err := file.Open()
			if !assert.NoError(t, err) {
				return
			}
			defer reader.Close()

			raw, err := ioutil.ReadAll(reader)
			if !assert.NoError(t, err) {
				return
			}

			assert.True(t, bytes.Equal(r.Raw, raw), "bytes doesn't match")
		}
	}
}

func TestLocalFile(t *testing.T) {
	local := LocalFile{Path: "testdata/lipsum.txt", FileMode: 755}

	assert.Equal(t, "lipsum.txt", local.Name())
	assert.Equal(t, os.FileMode(755), local.Mode())

	reader, err := local.Open()
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err := reader.Close()
		assert.NoError(t, err)
	}()

	content, err := ioutil.ReadAll(reader)
	if !assert.NoError(t, err) {
		return
	}

	raw, _ := ioutil.ReadFile("testdata/lipsum.txt")
	assert.Equal(t, raw, content)
}

func TestMemoryFile(t *testing.T) {
	raw := []byte("hello world")
	memory := MemoryFile{Path: "lipsum.txt", FileMode: 755, Raw: raw}

	assert.Equal(t, "lipsum.txt", memory.Name())
	assert.Equal(t, os.FileMode(755), memory.Mode())

	reader, err := memory.Open()
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err := reader.Close()
		assert.NoError(t, err)
	}()

	content, err := ioutil.ReadAll(reader)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, raw, content)
}
