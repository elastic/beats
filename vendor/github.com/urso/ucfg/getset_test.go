package ucfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGetPrimitives(t *testing.T) {
	c := New()

	c.SetBool("bool", -1, true)
	c.SetInt("int", -1, 42)
	c.SetUint("uint", -1, 12)
	c.SetFloat("float", -1, 2.3)
	c.SetString("str", -1, "abc")

	assert.True(t, c.HasField("bool"))
	assert.True(t, c.HasField("int"))
	assert.True(t, c.HasField("uint"))
	assert.True(t, c.HasField("float"))
	assert.True(t, c.HasField("str"))
	assert.Len(t, c.GetFields(), 5)

	path := c.Path(".")
	assert.Equal(t, "", path)

	cnt, err := c.CountField("bool")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("int")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("uint")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("float")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = c.CountField("str")
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	b, err := c.Bool("bool", -1)
	assert.NoError(t, err)
	assert.Equal(t, true, b)

	i, err := c.Int("int", -1)
	assert.NoError(t, err)
	assert.Equal(t, 42, int(i))

	u, err := c.Int("uint", -1)
	assert.NoError(t, err)
	assert.Equal(t, 12, int(u))

	f, err := c.Float("float", -1)
	assert.NoError(t, err)
	assert.Equal(t, 2.3, f)

	s, err := c.String("str", -1)
	assert.NoError(t, err)
	assert.Equal(t, "abc", s)
}

func TestSetGetChild(t *testing.T) {
	var err error
	c := New()
	child := New()

	child.SetInt("test", -1, 42)
	c.SetChild("child", -1, child)

	child, err = c.Child("child", -1)
	assert.Nil(t, err)

	i, err := child.Int("test", -1)
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "child", child.Path("."))
	assert.Equal(t, c, child.Parent())
}

func TestSetGetChildPath(t *testing.T) {
	c := New()

	err := c.SetInt("sub.test", -1, 42, PathSep("."))
	assert.NoError(t, err)

	sub, err := c.Child("sub", -1)
	assert.Nil(t, err)

	i, err := sub.Int("test", -1)
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	i, err = c.Int("sub.test", -1, PathSep("."))
	assert.Nil(t, err)
	assert.Equal(t, 42, int(i))

	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "sub", sub.Path("."))
	assert.Equal(t, c, sub.Parent())
}

func TestSetGetArray(t *testing.T) {
	c := New()

	child := New()
	child.SetInt("test", -1, 42)

	c.SetBool("a", 0, true)
	c.SetInt("a", 1, 42)
	c.SetFloat("a", 2, 3.14)
	c.SetString("a", 3, "string")
	c.SetUint("a", 4, 12)
	c.SetChild("a", 5, child)

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

	u, err := c.Uint("a", 4)
	assert.NoError(t, err)
	assert.Equal(t, 12, int(u))

	child, err = c.Child("a", 5)
	assert.Nil(t, err)
	assert.Equal(t, "", c.Path("."))
	assert.Equal(t, "a.5", child.Path("."))
}

func TestSetGetNestedPath(t *testing.T) {
	c := New()
	c.SetInt("a.1.b.0", -1, 23, PathSep("."))
	c.SetInt("a.0.b.1.c.2", -1, 42, PathSep("."))
	c.SetInt("", 0, 12, PathSep("."))

	i, err := c.Int("", 0, PathSep("."))
	assert.NoError(t, err)
	assert.Equal(t, 12, int(i))

	i, err = c.Int("a.1.b", 0, PathSep("."))
	assert.NoError(t, err)
	assert.Equal(t, 23, int(i))

	i, err = c.Int("a.0.b.1.c", 2, PathSep("."))
	assert.NoError(t, err)
	assert.Equal(t, 42, int(i))

	_, err = c.Int("a.2", -1, PathSep("."))
	assert.True(t, err != nil)

	_, err = c.Int("a", 2, PathSep("."))
	assert.True(t, err != nil)

	// manually walk up to "a.0.b.1.c.2"
	c, err = c.Child("a", 0, PathSep("."))
	assert.NoError(t, err)
	assert.NotNil(t, c)

	c, err = c.Child("b", 1, PathSep("."))
	assert.NoError(t, err)
	assert.NotNil(t, c)

	i, err = c.Int("c", 2, PathSep("."))
	assert.NoError(t, err)
	assert.Equal(t, 42, int(i))
}
