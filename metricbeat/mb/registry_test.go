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
	assert.NotNil(t, f, "factory function is nil")
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
	t.Run("without HostParser", func(t *testing.T) {
		registry := NewRegister()
		err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
		if err != nil {
			t.Fatal(err)
		}

		reg, err := registry.metricSetRegistration(moduleName, metricSetName)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, metricSetName, reg.Name)
		assert.NotNil(t, reg.Factory)
		assert.Nil(t, reg.HostParser)
		assert.False(t, reg.IsDefault)
		assert.Empty(t, reg.Namespace)
	})

	t.Run("with HostParser", func(t *testing.T) {
		registry := NewRegister()
		hostParser := func(Module, string) (HostData, error) { return HostData{}, nil }
		err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory, hostParser)
		if err != nil {
			t.Fatal(err)
		}

		reg, err := registry.metricSetRegistration(moduleName, metricSetName)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, reg.HostParser) // Can't compare functions in Go so just check for non-nil.
	})

	t.Run("with options HostParser", func(t *testing.T) {
		registry := NewRegister()
		hostParser := func(Module, string) (HostData, error) { return HostData{}, nil }
		err := registry.addMetricSet(moduleName, metricSetName, fakeMetricSetFactory, WithHostParser(hostParser))
		if err != nil {
			t.Fatal(err)
		}

		reg, err := registry.metricSetRegistration(moduleName, metricSetName)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, reg.HostParser) // Can't compare functions in Go so just check for non-nil.
	})

	t.Run("with namespace", func(t *testing.T) {
		const ns = moduleName + "foo.bar"

		registry := NewRegister()
		err := registry.addMetricSet(moduleName, metricSetName, fakeMetricSetFactory, WithNamespace(ns))
		if err != nil {
			t.Fatal(err)
		}

		reg, err := registry.metricSetRegistration(moduleName, metricSetName)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, metricSetName, reg.Name)
		assert.NotNil(t, reg.Factory)
		assert.Nil(t, reg.HostParser)
		assert.False(t, reg.IsDefault)
		assert.Equal(t, ns, reg.Namespace)
	})
}

func TestDefaultMetricSet(t *testing.T) {
	registry := NewRegister()
	err := registry.addMetricSet(moduleName, metricSetName, fakeMetricSetFactory, DefaultMetricSet())
	if err != nil {
		t.Fatal(err)
	}

	names, err := registry.DefaultMetricSets(moduleName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, names, metricSetName)
}

func TestMetricSetQuery(t *testing.T) {
	registry := NewRegister()
	err := registry.AddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)
	if err != nil {
		t.Fatal(err)
	}

	metricsets := registry.MetricSets(moduleName)
	assert.Equal(t, len(metricsets), 1)
	assert.Equal(t, metricsets[0], metricSetName)

	metricsets = registry.MetricSets("foo")
	assert.Equal(t, len(metricsets), 0)
}

func TestModuleQuery(t *testing.T) {
	registry := NewRegister()
	registry.modules[moduleName] = fakeModuleFactory

	modules := registry.Modules()
	assert.Equal(t, len(modules), 1)
	assert.Equal(t, modules[0], moduleName)
}
