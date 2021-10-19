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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewTestTree() *tree {
	defaultTemplate := template{
		Parts:     []string{"metric*"},
		Namespace: "foo",
		Delimiter: ".",
	}

	return NewTree(defaultTemplate)
}
func TestTreeInsert(t *testing.T) {
	test := NewTestTree()
	temp := template{
		Delimiter: "_",
		Namespace: "foo",
		Parts:     []string{"", "host", "metric*"},
	}
	test.Insert("test.localhost.*", temp)

	assert.Equal(t, len(test.root.children), 1)
	child := test.root.children["test"]
	assert.NotNil(t, child)
	assert.Nil(t, child.GetTemplate())

	cur := child
	assert.Equal(t, len(cur.children), 1)
	child = cur.children["localhost"]
	assert.NotNil(t, child)
	assert.Nil(t, child.GetTemplate())

	cur = child
	assert.Equal(t, len(cur.children), 1)
	child = cur.children["*"]
	assert.NotNil(t, child)
	assert.NotNil(t, child.GetTemplate())
	assert.Equal(t, &temp, child.GetTemplate())

	cur = child
	assert.Equal(t, len(cur.children), 0)
	test.Insert("test.localhost.*.foo", temp)
	assert.Equal(t, len(cur.children), 1)

	test.Insert("a.b.c.d", temp)
	assert.Equal(t, len(test.root.children), 2)
}

func TestTreeSearch(t *testing.T) {
	test := NewTestTree()
	temp := template{
		Delimiter: "_",
		Namespace: "foo",
		Parts:     []string{"", "host", "metric*"},
	}
	test.Insert("test.localhost.*", temp)

	// Search for a valid scenario
	outTemp := test.Search([]string{"test", "localhost", "bash", "stats"})
	assert.NotNil(t, outTemp)
	assert.Equal(t, outTemp, &temp)

	// Search for a case where only half the tree is traversed and there is no entry
	outTemp = test.Search([]string{"test"})
	assert.Nil(t, outTemp)

	// Search for a default case where root data is returned
	outTemp = test.Search([]string{"a.b.c.d"})
	assert.NotNil(t, outTemp)
	assert.Equal(t, outTemp, test.root.entry.value)
}

func TestTreeDelete(t *testing.T) {
	test := NewTestTree()
	temp := template{
		Delimiter: "_",
		Namespace: "foo",
		Parts:     []string{"", "host", "metric*"},
	}
	test.Insert("test.localhost.*", temp)
	test.Delete("test.localhost.*")

	assert.Equal(t, len(test.root.children), 0)

	test.Insert("test.localhost.*", temp)
	test.Insert("test.*", temp)
	test.Delete("test.*")

	assert.Equal(t, len(test.root.children), 1)
	assert.NotNil(t, test.root.FindChild("test"))

}
