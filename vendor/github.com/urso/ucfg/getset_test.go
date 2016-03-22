package ucfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGetPrimitives(t *testing.T) {
	c := New()

	c.SetBool("bool", 0, true)
	c.SetInt("int", 0, 42)
	c.SetFloat("float", 0, 2.3)
	c.SetString("str", 0, "abc")

	assert.True(t, c.HasField("bool"))
	assert.True(t, c.HasField("int"))
	assert.True(t, c.HasField("float"))
	assert.True(t, c.HasField("str"))
	assert.Len(t, c.GetFields(), 4)

	path := c.Path(".")
	assert.Equal(t, "", path)

	cnt, err := c.CountField("bool")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("int")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("float")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("str")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	b, err := c.Bool("bool", 0)
	assert.NoError(t, err)
	assert.Equal(t, true, b)

	i, err := c.Int("int", 0)
	assert.NoError(t, err)
	assert.Equal(t, 42, int(i))

	f, err := c.Float("float", 0)
	assert.NoError(t, err)
	assert.Equal(t, 2.3, f)

	s, err := c.String("str", 0)
	assert.NoError(t, err)
	assert.Equal(t, "abc", s)
}

func TestSetGetChild(t *testing.T) {
	var err error
	c := New()
	child := New()

	child.SetInt("test", 0, 42)
	c.SetChild("child", 0, child)

	child, err = c.Child("child", 0)
	assert.Nil(t, err)

	i, err := child.Int("test", 0)
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "child", child.Path("."))
	assert.Equal(t, c, child.Parent())
}

func TestSetGetChildPath(t *testing.T) {
	c := New()

	err := c.SetInt("sub.test", 0, 42, PathSep("."))
	assert.NoError(t, err)

	sub, err := c.Child("sub", 0)
	assert.Nil(t, err)

	i, err := sub.Int("test", 0)
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	i, err = c.Int("sub.test", 0, PathSep("."))
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "sub", sub.Path("."))
	assert.Equal(t, c, sub.Parent())
}

func TestSetGetArray(t *testing.T) {
	c := New()

	child := New()
	child.SetInt("test", 0, 42)

	c.SetBool("a", 0, true)
	c.SetInt("a", 1, 42)
	c.SetFloat("a", 2, 3.14)
	c.SetString("a", 3, "string")
	c.SetChild("a", 4, child)

	b, err := c.Bool("a", 0)
	assert.NoError(t, err)
	assert.Equal(t, true, b)

	i, err := c.Int("a", 1)
	assert.NoError(t, err)
	assert.Equal(t, 42, int(i))

	f, err := c.Float("a", 2)
	assert.NoError(t, err)
	assert.Equal(t, 3.14, f)

	s, err := c.String("a", 3)
	assert.NoError(t, err)
	assert.Equal(t, "string", s)

	child, err = c.Child("a", 4)
	assert.Nil(t, err)
	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "a.4", child.Path("."))
	assert.Equal(t, c, child.Parent())
}
