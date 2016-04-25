package ucfg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergePrimitives(t *testing.T) {
	c := New()
	c.SetBool("b", -1, true)
	c.SetInt("i", -1, 42)
	c.SetUint("u", -1, 23)
	c.SetFloat("f", -1, 3.14)
	c.SetString("s", -1, "string")

	c2 := newC()
	c2.SetBool("b", -1, true)
	c2.SetInt("i", -1, 42)
	c2.SetUint("u", -1, 23)
	c2.SetFloat("f", -1, 3.14)
	c2.SetString("s", -1, "string")

	tests := []interface{}{
		map[string]interface{}{
			"b": true,
			"i": 42,
			"u": 23,
			"f": 3.14,
			"s": "string",
		},
		node{
			"b": true,
			"i": 42,
			"u": 23,
			"f": 3.14,
			"s": "string",
		},
		struct {
			B bool
			I int
			U uint
			F float64
			S string
		}{true, 42, 23, 3.14, "string"},

		c,

		c2,
	}

	for i, in := range tests {
		t.Logf("run primitive test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		path := c.Path(".")
		assert.Equal(t, "", path)

		b, err := c.Bool("b", -1)
		assert.NoError(t, err)

		i, err := c.Int("i", -1)
		assert.NoError(t, err)

		u, err := c.Int("u", -1)
		assert.NoError(t, err)

		f, err := c.Float("f", -1)
		assert.NoError(t, err)

		s, err := c.String("s", -1)
		assert.NoError(t, err)

		assert.Equal(t, true, b)
		assert.Equal(t, 42, int(i))
		assert.Equal(t, 23, int(u))
		assert.Equal(t, 3.14, f)
		assert.Equal(t, "string", s)
	}
}

func TestMergeNested(t *testing.T) {
	sub := New()
	sub.SetBool("b", -1, true)

	c := New()
	c.SetChild("c", -1, sub)

	c2 := newC()
	c2.SetChild("c", -1, fromConfig(sub))

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
			"c": {"b": true},
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

		c2,
	}

	for i, in := range tests {
		t.Logf("merge nested test(%v): %+v", i, in)

		c := New()
		err := c.Merge(in)
		assert.NoError(t, err)

		sub, err := c.Child("c", -1)
		assert.NoError(t, err)

		b, err := sub.Bool("b", -1)
		assert.NoError(t, err)
		assert.True(t, b)

		assert.Equal(t, "", c.Path("."))
		assert.Equal(t, "c", sub.Path("."))
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

		sub, err := c.Child("c", -1)
		assert.NoError(t, err)
		if sub == nil {
			continue
		}

		b, err := sub.Bool("b", -1)
		assert.NoError(t, err)
		assert.True(t, b)

		assert.Equal(t, "", c.Path("."))
		assert.Equal(t, "c", sub.Path("."))
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
	sub.SetBool("b", -1, true)

	tests := []interface{}{
		map[string]interface{}{
			"a": []interface{}{
				true, 42, uint(23), 3.14, "string", sub,
			},
		},
		node{
			"a": []interface{}{
				true, 42, uint(23), 3.14, "string", sub,
			},
		},
		struct{ A []interface{} }{
			[]interface{}{
				true, 42, uint(23), 3.14, "string", sub,
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

		u, err := c.Uint("a", 2)
		assert.NoError(t, err)
		assert.Equal(t, 23, int(u))

		f, err := c.Float("a", 3)
		assert.NoError(t, err)
		assert.Equal(t, 3.14, f)

		s, err := c.String("a", 4)
		assert.NoError(t, err)
		assert.Equal(t, "string", s)

		sub, err := c.Child("a", 5)
		assert.NoError(t, err)
		b, err = sub.Bool("b", 0)
		assert.NoError(t, err)
		assert.Equal(t, true, b)

		assert.Equal(t, "", c.Path("."))
		assert.Equal(t, "a.5", sub.Path("."))
	}
}

func TestMergeChildArray(t *testing.T) {
	mk := func(i int) *Config {
		c := New()
		c.SetInt("i", -1, int64(i))
		return c
	}

	s1 := mk(1)
	s2 := mk(2)
	s3 := mk(3)

	arrConfig := []*Config{s1, s2, s3}
	arrC := []*C{fromConfig(s1), fromConfig(s2), fromConfig(s3)}
	arrIConfig := []interface{}{s1, s2, s3}
	arrIC := []interface{}{fromConfig(s1), fromConfig(s2), fromConfig(s3)}

	tests := []interface{}{
		map[string]interface{}{"a": arrIConfig},
		map[string]interface{}{"a": arrIC},

		map[string]interface{}{"a": arrConfig},
		map[string]interface{}{"a": arrC},

		node{"a": arrIConfig},
		node{"a": arrIC},

		node{"a": arrConfig},
		node{"a": arrC},

		struct{ A []interface{} }{arrIConfig},
		struct{ A []interface{} }{arrIC},
		struct{ A []*Config }{A: arrConfig},
		struct{ A []*C }{arrC},
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

			assert.Equal(t, "", c.Path("."))
			assert.Equal(t, fmt.Sprintf("a.%v", i), sub.Path("."))
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

		b, err := c.Bool("b", -1)
		assert.NoError(t, err)
		assert.Equal(t, true, b)
	}
}

func TestMergeArrayPatterns(t *testing.T) {
	tests := []interface{}{
		node{
			"object": node{
				"sub": node{
					"0": node{"title": "test0"},
					"1": node{"title": "test1"},
					"2": node{"title": "test2"},
				},
			},
		},

		node{
			"object": node{
				"sub": []node{
					{"title": "test0"},
					{"title": "test1"},
					{"title": "test2"},
				},
			},
		},

		node{
			"object.sub": []node{
				{"title": "test0"},
				{"title": "test1"},
				{"title": "test2"},
			},
		},

		node{
			"object.sub.0.title": "test0",
			"object.sub.1.title": "test1",
			"object.sub.2.title": "test2",
		},
	}

	for i, test := range tests {
		t.Logf("test (%v): %v", i, test)
		c, err := NewFrom(test, PathSep("."))
		if err != nil {
			t.Fatal(err)
		}

		for x := 0; x < 3; x++ {
			s, err := c.String(fmt.Sprintf("object.sub.%v.title", x), -1, PathSep("."))
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("test%v", x), s)
		}
	}
}
