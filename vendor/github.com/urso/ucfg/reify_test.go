package ucfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpackPrimitiveValues(t *testing.T) {
	tests := []interface{}{
		New(),
		&map[string]interface{}{},
		map[string]interface{}{},
		node{},
		&node{},
		&struct {
			B bool
			I int
			F float64
			S string
		}{},
		&struct {
			B interface{}
			I interface{}
			F interface{}
			S interface{}
		}{},
		&struct {
			B *bool
			I *int
			F *float64
			S *string
		}{},
	}

	c, _ := NewFrom(node{
		"b": true,
		"i": 42,
		"f": 3.14,
		"s": "string",
	})

	for i, out := range tests {
		t.Logf("test unpack primitives(%v) into: %v", i, out)
		err := c.Unpack(out)
		if err != nil {
			t.Fatalf("failed to unpack: %v", err)
		}
	}

	// validate content by merging struct
	for i, in := range tests {
		t.Logf("test unpack primitives(%v) check: %v", i, in)

		c, err := NewFrom(in)
		if err != nil {
			t.Errorf("failed")
			continue
		}

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

func TestUnpackNested(t *testing.T) {
	var genSub = func(name string) *Config {
		s := New()
		s.SetBool(name, 0, false)
		return s
	}

	sub, _ := NewFrom(node{"b": true})
	c, _ := NewFrom(node{"c": sub})

	t.Logf("sub: %v", sub)
	t.Logf("c: %v", c)

	tests := []interface{}{
		New(),

		map[string]interface{}{},
		map[string]*Config{},
		map[string]map[string]bool{},
		map[string]map[string]interface{}{},
		map[string]interface{}{
			"c": map[string]interface{}{
				"b": false,
			},
		},
		map[string]interface{}{
			"c": nil,
		},
		map[string]*Config{
			"c": nil,
		},
		map[string]interface{}{
			"c": New(),
		},
		map[string]interface{}{
			"c": genSub("b"),
		},
		map[string]interface{}{
			"c": genSub("d"),
		},
		map[string]*struct{ B bool }{},
		map[string]*struct{ B bool }{"c": nil},
		map[string]struct{ B bool }{},

		node{},
		node{"c": node{}},
		node{"c": node{"b": false}},
		node{"c": genSub("d")},

		&struct{ C *Config }{},
		&struct{ C *Config }{sub},
		&struct{ C *Config }{genSub("d")},
		&struct{ C map[string]interface{} }{},
		&struct{ C node }{},
		&struct{ C struct{ B bool } }{},
		&struct{ C *struct{ B bool } }{&struct{ B bool }{}},
		&struct{ C *struct{ B bool } }{},
	}

	for i, out := range tests {
		t.Logf("test unpack nested(%v) into: %v", i, out)
		err := c.Unpack(out)
		if err != nil {
			t.Fatalf("failed to unpack: %v", err)
		}
	}

	// validate content by merging struct
	for i, in := range tests {
		t.Logf("test unpack nested(%v) check: %v", i, in)

		c, err := NewFrom(in)
		if err != nil {
			t.Errorf("failed")
			continue
		}

		sub, err := c.Child("c", 0)
		assert.NoError(t, err)

		b, err := sub.Bool("b", 0)
		assert.NoError(t, err)
		assert.True(t, b)
	}
}

func TestUnpackNestedPath(t *testing.T) {
	tests := []interface{}{
		&struct {
			B bool `config:"c.b"`
		}{},

		&struct {
			B interface{} `config:"c.b"`
		}{},
	}

	sub, _ := NewFrom(node{"b": true})
	c, _ := NewFrom(node{"c": sub})

	for i, out := range tests {
		t.Logf("test unpack nested path(%v) into: %v", i, out)
		err := c.Unpack(out, PathSep("."))
		if err != nil {
			t.Fatalf("failed to unpack: %v", err)
		}
	}

	// validate content by merging struct (unnested)
	for i, in := range tests {
		t.Logf("test unpack nested(%v) check: %v", i, in)

		c, err := NewFrom(in)
		if err != nil {
			t.Errorf("failed")
			continue
		}

		b, err := c.Bool("c.b", 0)
		assert.NoError(t, err)
		assert.True(t, b)
	}
}

func TestUnpackArray(t *testing.T) {
	c, _ := NewFrom(node{"a": []int{1, 2, 3}})

	tests := []interface{}{
		map[string]interface{}{},
		map[string]interface{}{
			"a": []int{},
		},
		map[string][]int{
			"a": []int{},
		},
		map[string]interface{}{
			"a": []interface{}{},
		},
		map[string][]int{},

		node{},
		node{
			"a": []int{},
		},
		node{
			"a": []interface{}{},
		},

		&struct{ A []int }{},
		&struct{ A []uint }{},
		&struct{ A []interface{} }{},
		&struct{ A interface{} }{},
		&struct{ A [3]int }{},
		&struct{ A [3]uint }{},
		&struct{ A [3]interface{} }{},
	}

	for i, out := range tests {
		t.Logf("test unpack array(%v) into: %v", i, out)
		err := c.Unpack(out)
		if err != nil {
			t.Fatalf("failed to unpack: %v", err)
		}
	}

	// validate content by merging struct
	for i, in := range tests {
		t.Logf("test unpack nested(%v) check: %v", i, in)

		c, err := NewFrom(in)
		if err != nil {
			t.Errorf("failed")
			continue
		}

		for i := 0; i < 3; i++ {
			v, err := c.Int("a", i)
			assert.NoError(t, err)
			assert.Equal(t, i+1, int(v))
		}
	}
}

func TestUnpackInline(t *testing.T) {
	type SubType struct{ B bool }
	type SubInterface struct{ B interface{} }

	tests := []interface{}{
		&struct {
			C SubType `config:",inline"`
		}{SubType{true}},
		&struct {
			SubType `config:",inline"`
		}{SubType{true}},

		&struct {
			C SubInterface `config:",inline"`
		}{SubInterface{true}},
		&struct {
			SubInterface `config:",inline"`
		}{SubInterface{true}},

		&struct {
			C map[string]bool `config:",inline"`
		}{map[string]bool{"b": true}},

		&struct {
			C map[string]interface{} `config:",inline"`
		}{map[string]interface{}{"b": true}},

		&struct {
			C node `config:",inline"`
		}{node{"b": true}},
	}

	c, _ := NewFrom(map[string]bool{"b": true})

	for i, out := range tests {
		t.Logf("test unpack with inline(%v) into: %v", i, out)
		err := c.Unpack(out)
		if err != nil {
			t.Fatalf("failed to unpack: %v", err)
		}
	}

	// validate content by merging struct
	for i, in := range tests {
		t.Logf("test unpack inlined(%v) check: %v", i, in)

		c, err := NewFrom(in)
		if err != nil {
			t.Fatalf("failed with: %v", err)
		}

		b, err := c.Bool("b", 0)
		assert.NoError(t, err)
		assert.Equal(t, true, b)
	}
}
