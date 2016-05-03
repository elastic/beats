// +build !integration

package mb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	moduleName    = "mymodule"
	metricSetName = "mymetricset"
)

var fakeModuleFactory = func(b BaseModule) (Module, error) { return nil, nil }
var fakeMetricSetFactory = func(b BaseMetricSet) (MetricSet, error) { return nil, nil }

func TestAddModuleEmptyName(t *testing.T) {
	registry := NewRegister()
	err := registry.AddModule("", fakeModuleFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "module name is required", err.Error())
	}
}

func TestAddModuleNilFactory(t *testing.T) {
	registry := NewRegister()
	err := registry.AddModule(moduleName, nil)
	if assert.Error(t, err) {
		assert.Equal(t, "module 'mymodule' cannot be registered with a nil factory", err.Error())
	}
}

func TestAddModuleDuplicateName(t *testing.T) {
	registry := NewRegister()
	err := registry.AddModule(moduleName, fakeModuleFactory)
	if err != nil {
		t.Fatal(err)
	}

	err = registry.AddModule(moduleName, fakeModuleFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "module 'mymodule' is already registered", err.Error())
	}
}

func TestAddModule(t *testing.T) {
	registry := NewRegister()
	err := registry.AddModule(moduleName, fakeModuleFactory)
	if err != nil {
		t.Fatal(err)
	}
	factory, found := registry.modules[moduleName]
	assert.True(t, found, "module not found")
	assert.NotNil(t, factory, "factory fuction is nil")
}

func TestAddMetricSetEmptyModuleName(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet("", metricSetName, fakeMetricSetFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "module name is required", err.Error())
	}
}

func TestAddMetricSetEmptyName(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, "", fakeMetricSetFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "metricset name is required", err.Error())
	}
}

func TestAddMetricSetNilFactory(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, metricSetName, nil)
	if assert.Error(t, err) {
		assert.Equal(t, "metricset 'mymodule/mymetricset' cannot be registered with a nil factory", err.Error())
	}
}

func TestAddMetricSetDuplicateName(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
	if err != nil {
		t.Fatal(err)
	}

	err = registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "metricset 'mymodule/mymetricset' is already registered", err.Error())
	}
}

func TestAddMetricSet(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
	if err != nil {
		t.Fatal(err)
	}
	f, found := registry.metricSets[moduleName][metricSetName]
	assert.True(t, found, "metricset not found")
	assert.NotNil(t, f, "factory fuction is nil")
}

func TestModuleFactory(t *testing.T) {
	registry := NewRegister()
	registry.modules[moduleName] = fakeModuleFactory

	module := registry.moduleFactory(moduleName)
	assert.NotNil(t, module)
}

func TestModuleFactoryUnknownModule(t *testing.T) {
	registry := NewRegister()
	module := registry.moduleFactory("unknown")
	assert.Nil(t, module)
}

func TestMetricSetFactory(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
	if err != nil {
		t.Fatal(err)
	}

	ms, err := registry.metricSetFactory(moduleName, metricSetName)
	if assert.NoError(t, err) {
		assert.NotNil(t, ms)
	}
}
