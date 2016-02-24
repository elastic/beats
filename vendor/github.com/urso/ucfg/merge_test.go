package ucfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergePrimitives(t *testing.T) {
	c := New()
	c.SetBool("b", 0, true)
	c.SetInt("i", 0, 42)
	c.SetFloat("f", 0, 3.14)
	c.SetString("s", 0, "string")

	tests := []interface{}{
		map[string]interface{}{
			"b": true,
			"i": 42,
			"f": 3.14,
			"s": "string",
		},
		node{
			"b": true,
			"i": 42,
			"f": 3.14,
			"s": "string",
		},
		struct {
			B bool
			I int
			F float64
			S string
		}{true, 42, 3.14, "string"},
		c,
	}

	for i, in := range tests {
		t.Logf("run primitive test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		b, err := c.Bool("b", 0)
		assert.NoError(t, err)

		i, err := c.Int("i", 0)
		assert.NoError(t, err)

		f, err := c.Float("f", 0)
		assert.NoError(t, err)

		s, err := c.String("s", 0)
		assert.NoError(t, err)

		assert.Equal(t, true, b)
		assert.Equal(t, 42, int(i))
		assert.Equal(t, 3.14, f)
		assert.Equal(t, "string", s)
	}
}

func TestMergeNested(t *testing.T) {
	sub := New()
	sub.SetBool("b", 0, true)

	c := New()
	c.SetChild("c", 0, sub)

	tests := []interface{}{
		map[string]interface{}{
			"c": map[string]interface{}{
				"b": true,
			},
		},
		map[string]*Config{
			"c": sub,
		},
		map[string]map[string]bool{
			"c": map[string]bool{
				"b": true,
			},
		},

		node{"c": map[string]interface{}{"b": true}},
		node{"c": map[string]bool{"b": true}},
		node{"c": node{"b": true}},
		node{"c": struct{ B bool }{true}},
		node{"c": sub},

		struct{ C map[string]interface{} }{
			map[string]interface{}{"b": true},
		},
		struct{ C map[string]bool }{
			map[string]bool{"b": true},
		},
		struct{ C node }{
			node{"b": true},
		},
		struct{ C *Config }{sub},
		struct{ C struct{ B bool } }{struct{ B bool }{true}},
		struct{ C interface{} }{struct{ B bool }{true}},
		struct{ C interface{} }{struct{ B interface{} }{true}},
		struct{ C struct{ B interface{} } }{struct{ B interface{} }{true}},

		c,
	}

	for i, in := range tests {
		t.Logf("merge nested test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		sub, err := c.Child("c", 0)
		assert.NoError(t, err)

		b, err := sub.Bool("b", 0)
		assert.NoError(t, err)

		assert.True(t, b)
	}
}

func TestMergeNestedPath(t *testing.T) {
	tests := []interface{}{
		map[string]interface{}{
			"c.b": true,
		},

		map[string]bool{
			"c.b": true,
		},

		node{
			"c.b": true,
		},

		struct {
			B bool `config:"c.b"`
		}{true},
	}

	for i, in := range tests {
		t.Logf("merge nested test(%v), %+v", i, in)

		c := New()
		err := c.Merge(in, PathSep("."))

		assert.NoError(t, err)

		sub, err := c.Child("c", 0)
		assert.NoError(t, err)

		b, err := sub.Bool("b", 0)
		assert.NoError(t, err)

		assert.True(t, b)
	}
}

func TestMergeArray(t *testing.T) {
	tests := []interface{}{
		map[string]interface{}{
			"a": []interface{}{1, 2, 3},
		},
		map[string]interface{}{
			"a": []int{1, 2, 3},
		},

		node{
			"a": []int{1, 2, 3},
		},

		struct{ A []interface{} }{[]interface{}{1, 2, 3}},
		struct{ A []int }{[]int{1, 2, 3}},
	}

	for i, in := range tests {
		t.Logf("merge mixed array test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		for i := 0; i < 3; i++ {
			v, err := c.Int("a", i)
			assert.NoError(t, err)
			assert.Equal(t, i+1, int(v))
		}
	}
}

func TestMergeMixedArray(t *testing.T) {
	sub := New()
	sub.SetBool("b", 0, true)

	tests := []interface{}{
		map[string]interface{}{
			"a": []interface{}{
				true, 42, 3.14, "string", sub,
			},
		},
		node{
			"a": []interface{}{
				true, 42, 3.14, "string", sub,
			},
		},
		struct{ A []interface{} }{
			[]interface{}{
				true, 42, 3.14, "string", sub,
			},
		},
	}

	for i, in := range tests {
		t.Logf("merge mixed array test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

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

		sub, err := c.Child("a", 4)
		assert.NoError(t, err)
		b, err = sub.Bool("b", 0)
		assert.NoError(t, err)
		assert.Equal(t, true, b)
	}
}

func TestMergeChildArray(t *testing.T) {
	mk := func(i int) *Config {
		c := New()
		c.SetInt("i", 0, int64(i))
		return c
	}

	s1 := mk(1)
	s2 := mk(2)
	s3 := mk(3)

	tests := []interface{}{
		map[string]interface{}{
			"a": []interface{}{s1, s2, s3},
		},
		map[string]interface{}{
			"a": []*Config{s1, s2, s3},
		},
		node{
			"a": []*Config{s1, s2, s3},
		},

		struct{ A []interface{} }{[]interface{}{s1, s2, s3}},
		struct{ A []*Config }{[]*Config{s1, s2, s3}},
	}

	for i, in := range tests {
		t.Logf("merge mixed array test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		for i := 0; i < 3; i++ {
			sub, err := c.Child("a", i)
			assert.NoError(t, err)

			v, err := sub.Int("i", 0)
			assert.NoError(t, err)
			assert.Equal(t, i+1, int(v))
		}
	}
}

func TestMergeSquash(t *testing.T) {
	type subType struct{ B bool }
	type subInterface struct{ B interface{} }

	tests := []interface{}{
		&struct {
			C subType `config:",squash"`
		}{subType{true}},
		&struct {
			subType `config:",squash"`
		}{subType{true}},

		&struct {
			C subInterface `config:",squash"`
		}{subInterface{true}},
		&struct {
			subInterface `config:",squash"`
		}{subInterface{true}},

		&struct {
			C map[string]bool `config:",squash"`
		}{map[string]bool{"b": true}},

		&struct {
			C map[string]interface{} `config:",squash"`
		}{map[string]interface{}{"b": true}},

		&struct {
			C node `config:",squash"`
		}{node{"b": true}},
		&struct {
			node `config:",squash"`
		}{node{"b": true}},
	}

	for i, in := range tests {
		t.Logf("merge squash test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		b, err := c.Bool("b", 0)
		assert.NoError(t, err)
		assert.Equal(t, true, b)
	}
}
