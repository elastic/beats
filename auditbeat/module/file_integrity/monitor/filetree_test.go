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

package monitor

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type visitParams struct {
	path  string
	isDir bool
}

func init() {
	PathSeparator = "/"
}

func TestVisit(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/usr/bin/python"))
	assertNoError(t, tree.AddFile("/usr/bin/tar"))
	assertNoError(t, tree.AddFile("/usr/lib/libz.a"))
	assertNoError(t, tree.AddDir("/tmp/"))

	for testIdx, testData := range []struct {
		dir    string
		result []string
		isDir  []bool
	}{
		{"/",
			[]string{"/", "/tmp", "/usr", "/usr/bin", "/usr/bin/python", "/usr/bin/tar", "/usr/lib", "/usr/lib/libz.a"},
			[]bool{true, true, true, true, false, false, true, false}},
		{"/usr",
			[]string{"/usr", "/usr/bin", "/usr/bin/python", "/usr/bin/tar", "/usr/lib", "/usr/lib/libz.a"},
			[]bool{true, true, false, false, true, false}},
		{"/usr/bin",
			[]string{"/usr/bin", "/usr/bin/python", "/usr/bin/tar"},
			[]bool{true, false, false}},
		{"/usr/lib",
			[]string{"/usr/lib", "/usr/lib/libz.a"},
			[]bool{true, false}},
		{"/tmp/",
			[]string{"/tmp"},
			[]bool{true}},
		{"/usr/bin/python",
			[]string{"/usr/bin/python"},
			[]bool{false}},
	} {
		for _, order := range []VisitOrder{PreOrder, PostOrder} {
			failMsg := fmt.Sprintf("test entry %d for path '%s' order:%v", testIdx, testData.dir, order)
			calls := map[string]bool{}
			ncalls := 0

			err := tree.Visit(testData.dir, order, func(path string, isDir bool) error {
				calls[path] = isDir
				ncalls++
				return nil
			})
			assertNoError(t, err)

			assert.Equal(t, len(testData.result), ncalls, failMsg)
			assert.Equal(t, len(testData.result), len(calls), failMsg)
			keys := make([]string, len(calls))
			flags := make([]bool, len(calls))
			i := 0
			for k := range calls {
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			for idx, val := range keys {
				flags[idx] = calls[val]
			}
			assert.Equal(t, testData.result, keys, failMsg)
			assert.Equal(t, testData.isDir, flags, failMsg)
		}
	}
}

func TestVisitOrder(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/a/b/file"))

	expected := []visitParams{
		{"/", true},
		{"/a", true},
		{"/a/b", true},
		{"/a/b/file", false},
	}

	for _, order := range []VisitOrder{PreOrder, PostOrder} {
		var result []visitParams
		err := tree.Visit("/", order, func(path string, isDir bool) error {
			result = append(result, visitParams{path, isDir})
			return nil
		})
		assertNoError(t, err)

		assert.Equal(t, expected, result)

		for a, b := 0, len(expected)-1; a < b; a++ {
			expected[a], expected[b] = expected[b], expected[a]
			b--
		}
	}
}

func TestVisitError(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/foo"))
	assert.Error(t, tree.Visit("/bar", PreOrder, func(path string, isDir bool) error {
		return nil
	}))
}

func TestVisitCancel(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/a/b/file"))

	myErr := errors.New("some error")

	for idx, test := range []struct {
		order    VisitOrder
		cancel   string
		expected []visitParams
	}{
		{PreOrder, "/a", []visitParams{
			{"/", true}}},
		{PostOrder, "/a", []visitParams{
			{"/a/b/file", false},
			{"/a/b", true}}},
		{PreOrder, "/a/b/file", []visitParams{
			{"/", true},
			{"/a", true},
			{"/a/b", true}}},
	} {
		failMsg := fmt.Sprintf("test at index %d", idx)
		var result []visitParams
		err := tree.Visit("/", test.order, func(path string, isDir bool) error {
			if path == test.cancel {
				return myErr
			}
			result = append(result, visitParams{path, isDir})
			return nil
		})
		assert.Equal(t, myErr, err, failMsg)
		assert.Equal(t, test.expected, result, failMsg)
	}
}

func TestFilesAsDir(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/foo"))
	assert.Error(t, tree.AddFile("/foo/bar"))
	assert.Error(t, tree.Visit("/foo/bar", PreOrder, func(path string, isDir bool) error {
		return nil
	}))
}

func TestRemove(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/usr/bin/python"))
	assertNoError(t, tree.AddFile("/usr/bin/tar"))
	assertNoError(t, tree.AddFile("/usr/lib/libz.a"))
	assertNoError(t, tree.AddDir("/tmp/"))

	assertNoError(t, tree.Remove("/tmp"))
	assertNoError(t, tree.Remove("/usr/lib"))
	assertNoError(t, tree.Remove("/usr/bin/python"))

	expected := []visitParams{
		{"/", true},
		{"/usr", true},
		{"/usr/bin", true},
		{"/usr/bin/tar", false},
	}

	var result []visitParams
	err := tree.Visit("/", PreOrder, func(path string, isDir bool) error {
		result = append(result, visitParams{path, isDir})
		return nil
	})
	assertNoError(t, err)
	assert.Equal(t, expected, result)

	assert.Error(t, tree.Remove("/usr/src/linux"))
}

func TestAt(t *testing.T) {
	tree := FileTree{}
	assertNoError(t, tree.AddFile("/usr/bin/python"))
	assertNoError(t, tree.AddFile("/usr/bin/tar"))
	assertNoError(t, tree.AddFile("/usr/lib/libz.a"))
	assertNoError(t, tree.AddDir("/tmp/"))

	expected := []visitParams{
		{"/", true},
		{"/bin", true},
		{"/bin/python", false},
		{"/bin/tar", false},
		{"/lib", true},
		{"/lib/libz.a", false},
	}

	var result []visitParams
	subtree, err := tree.At("/usr")
	assertNoError(t, err)
	err = subtree.Visit("/", PostOrder, func(path string, isDir bool) error {
		result = append(result, visitParams{path, isDir})
		return nil
	})
	assertNoError(t, err)

	sort.Slice(result, func(i, j int) bool { return result[i].path < result[j].path })
	assert.Equal(t, expected, result)
}
