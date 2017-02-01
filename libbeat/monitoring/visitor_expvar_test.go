// +build !integration

package monitoring

import (
	"expvar"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIterExpvarIgnoringMonitoringVars(t *testing.T) {
	vars := map[string]int64{
		"sub.registry.v1": 1,
		"sub.registry.v2": 2,
		"v3":              3,
	}
	collected := map[string]int64{}

	reg := NewRegistry(PublishExpvar)
	for name, v := range vars {
		i := NewInt(reg, name, Report)
		i.Add(v)
	}

	DoExpvars(func(name string, v interface{}) {
		if _, exists := vars[name]; exists {
			collected[name] = v.(int64)
		}
	})
	assert.Equal(t, map[string]int64{}, collected)
}

func TestIterExpvarCaptureVars(t *testing.T) {
	i := getOrCreateInt("test.integer")
	i.Set(42)

	s := getOrCreateString("test.string")
	s.Set("testing")

	var m *expvar.Map
	if v := expvar.Get("test.map"); v != nil {
		m = v.(*expvar.Map)
	} else {
		m = expvar.NewMap("test.map")
		m.Add("i1", 1)
		m.Add("i2", 2)
	}

	expected := map[string]interface{}{
		"test.integer": int64(42),
		"test.string":  "testing",
		"test.map.i1":  int64(1),
		"test.map.i2":  int64(2),
	}

	collected := map[string]interface{}{}
	DoExpvars(func(name string, v interface{}) {
		if _, exists := expected[name]; exists {
			collected[name] = v
		}
	})

	assert.Equal(t, collected, expected)
}

func getOrCreateInt(name string) *expvar.Int {
	if v := expvar.Get(name); v != nil {
		return v.(*expvar.Int)
	}
	return expvar.NewInt(name)
}

func getOrCreateString(name string) *expvar.String {
	if v := expvar.Get(name); v != nil {
		return v.(*expvar.String)
	}
	return expvar.NewString(name)
}
