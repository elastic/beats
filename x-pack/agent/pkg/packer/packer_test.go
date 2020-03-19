// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packer

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacker(t *testing.T) {
	type tt struct {
		content  map[string]string
		patterns []string
		failed   bool
	}

	withFiles := func(test tt, fn func(pattern []string, t *testing.T)) func(t *testing.T) {
		return func(t *testing.T) {
			d, err := ioutil.TempDir("", "packer")
			require.NoError(t, err)
			defer os.RemoveAll(d)

			for f, v := range test.content {
				path := filepath.Join(d, f)
				err := ioutil.WriteFile(path, []byte(v), 0666)
				require.NoError(t, err)
			}

			patterns := make([]string, len(test.patterns))
			for i, p := range test.patterns {
				patterns[i] = filepath.Join(d, p)
			}

			fn(patterns, t)
		}
	}

	normalize := func(m PackMap) map[string]string {
		newM := make(map[string]string, len(m))
		for k, v := range m {
			newM[filepath.Base(k)] = string(v)
		}
		return newM
	}

	testcases := map[string]tt{
		"single files": {
			content: map[string]string{
				"abc.txt": "hello world",
			},
			patterns: []string{"abc.txt"},
		},
		"multiples files": {
			content: map[string]string{
				"abc.txt":  "hello world",
				"abc2.txt": "another content",
			},
			patterns: []string{"abc.txt", "abc2.txt"},
		},
		"multiples files with wildcards": {
			content: map[string]string{
				"abc.txt":  "hello world",
				"abc2.txt": "another \n\rcontent",
			},
			patterns: []string{"abc*"},
		},
		"duplicate files": {
			content: map[string]string{
				"abc.txt": "hello world",
			},
			patterns: []string{"abc.txt", "abc.txt"},
			failed:   true,
		},
		"large file": {
			content: map[string]string{
				"abc.txt": mustRandStr(1024 * 1014 * 2),
			},
			patterns: []string{"abc.txt"},
		},
	}

	for name, test := range testcases {
		t.Run(name, withFiles(test, func(patterns []string, t *testing.T) {
			packed, files, err := Pack(patterns...)
			if test.failed {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.failed, err != nil)

			uncompressed, err := Unpack(packed)
			assert.NoError(t, err)

			norm := normalize(uncompressed)
			assert.Equal(t, len(norm), len(files))
			assert.True(t, reflect.DeepEqual(test.content, norm))
		}))
	}
}

func randStr(length int) (string, error) {
	r := make([]byte, length)
	_, err := rand.Read(r)

	if err != nil {
		return "", err
	}

	return string(r), nil
}

func mustRandStr(l int) string {
	s, err := randStr(l)
	if err != nil {
		panic(err)
	}
	return s
}
