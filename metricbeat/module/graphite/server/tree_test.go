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
