// +build !integration

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryEmpty(t *testing.T) {
	defer Clear()

	// get value
	v := Get("missing")
	if v != nil {
		t.Errorf("got %v, wanted nil", v)
	}

	// get value with recursive lookup
	v = Get("missing.value")
	if v != nil {
		t.Errorf("got %v, wanted nil", v)
	}

	// get missing registry
	reg := GetRegistry("missing")
	if reg != nil {
		t.Errorf("got %v, wanted nil", reg)
	}

	// get registry with recursive lookup
	reg = GetRegistry("missing.registry")
	if reg != nil {
		t.Errorf("got %v, wanted nil", reg)
	}
}

func TestRegistryGet(t *testing.T) {
	defer Clear()

	name1 := "v"
	nameSub1 := "sub.registry1"
	nameSub2 := "sub.registry2"
	name2 := nameSub1 + "." + name1
	name3 := nameSub2 + "." + name1

	// register top-level and recursive metric
	v1 := NewInt(Default, name1, Report)
	sub1 := Default.NewRegistry(nameSub1)
	sub2 := Default.NewRegistry(nameSub2)
	v2 := NewString(nil, name2, Report)
	v3 := NewFloat(sub2, name1, Report)

	// get values
	v := Get(name1)
	assert.Equal(t, v, v1)

	// get nested metric from top-level
	v = Get(name2)
	assert.Equal(t, v, v2)
	v = Get(name3)
	assert.Equal(t, v, v3)

	// get sub registry
	reg1 := GetRegistry(nameSub1)
	assert.Equal(t, sub1, reg1)
	reg2 := GetRegistry(nameSub2)
	assert.Equal(t, sub2, reg2)

	// get value from sub-registry
	v = reg1.Get(name1)
	assert.Equal(t, v, v2)

	v = reg2.Get(name1)
	assert.Equal(t, v, v3)
}

func TestRegistryRemove(t *testing.T) {
	defer Clear()

	name1 := "v"
	nameSub1 := "sub.registry1"
	nameSub2 := "sub.registry2"
	name2 := nameSub1 + "." + name1
	name3 := nameSub2 + "." + name1

	// register top-level and recursive metric
	NewInt(Default, name1, Report)
	sub1 := Default.NewRegistry(nameSub1)
	sub2 := Default.NewRegistry(nameSub2)
	NewInt(Default, name2, Report)
	NewInt(sub2, name1, Report)

	// remove metrics:
	Remove(name1)
	sub1.Remove(name1) // == Remove(name2)
	Remove(name3)      // remove name 3 recursively

	// check no variable is reachable
	assert.Nil(t, Get(name1))
	assert.Nil(t, Get(name2))
	assert.Nil(t, Get(name3))
}

func TestRegistryIter(t *testing.T) {
	defer Clear()

	vars := map[string]int64{
		"sub.registry.v1": 1,
		"sub.registry.v2": 2,
		"v3":              3,
	}

	for name, v := range vars {
		i := NewInt(Default, name, Report)
		i.Add(v)
	}

	collected := map[string]int64{}
	Do(Full, func(name string, v interface{}) {
		collected[name] = v.(int64)
	})

	assert.Equal(t, vars, collected)
}
