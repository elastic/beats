// +build !integration

package helper

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const (
	moduleName    = "mymodule"
	metricSetName = "mymetricset"
)

func TestAddModulerEmptyName(t *testing.T) {
	registry := &Register{}
	err := registry.AddModuler("", func() Moduler { return nil })
	if assert.Error(t, err) {
		assert.Equal(t, "module name is required", err.Error())
	}
}

func TestAddModulerNilFactory(t *testing.T) {
	registry := &Register{}
	err := registry.AddModuler(moduleName, nil)
	if assert.Error(t, err) {
		assert.Equal(t, "module 'mymodule' cannot be registered with a nil factory", err.Error())
	}
}

func TestAddModulerDuplicateName(t *testing.T) {
	registry := &Register{}
	err := registry.AddModuler(moduleName, func() Moduler { return nil })
	if err != nil {
		t.Fatal(err)
	}

	err = registry.AddModuler(moduleName, func() Moduler { return nil })
	if assert.Error(t, err) {
		assert.Equal(t, "module 'mymodule' is already registered", err.Error())
	}
}

func TestAddModuler(t *testing.T) {
	registry := &Register{}
	err := registry.AddModuler(moduleName, func() Moduler { return nil })
	if err != nil {
		t.Fatal(err)
	}
	factory, found := registry.Modulers[moduleName]
	assert.True(t, found, "module not found")
	assert.NotNil(t, factory, "factory fuction is nil")
}

func TestAddMetricSeterEmptyModuleName(t *testing.T) {
	registry := &Register{}
	err := registry.AddMetricSeter("", metricSetName, func() MetricSeter { return nil })
	if assert.Error(t, err) {
		assert.Equal(t, "module name is required", err.Error())
	}
}

func TestAddMetricSeterEmptyName(t *testing.T) {
	registry := &Register{}
	err := registry.AddMetricSeter(moduleName, "", func() MetricSeter { return nil })
	if assert.Error(t, err) {
		assert.Equal(t, "metricset name is required", err.Error())
	}
}

func TestAddMetricSeterNilFactory(t *testing.T) {
	registry := &Register{}
	err := registry.AddMetricSeter(moduleName, metricSetName, nil)
	if assert.Error(t, err) {
		assert.Equal(t, "metricset 'mymodule/mymetricset' cannot be registered with a nil factory", err.Error())
	}
}

func TestAddMetricSeterDuplicateName(t *testing.T) {
	registry := &Register{}
	factory := func() MetricSeter { return nil }
	err := registry.AddMetricSeter(moduleName, metricSetName, factory)
	if err != nil {
		t.Fatal(err)
	}

	err = registry.AddMetricSeter(moduleName, metricSetName, factory)
	if assert.Error(t, err) {
		assert.Equal(t, "metricset 'mymodule/mymetricset' is already registered", err.Error())
	}
}

func TestAddMetricSeter(t *testing.T) {
	registry := &Register{}
	factory := func() MetricSeter { return nil }
	err := registry.AddMetricSeter(moduleName, metricSetName, factory)
	if err != nil {
		t.Fatal(err)
	}
	f, found := registry.MetricSeters[moduleName][metricSetName]
	assert.True(t, found, "metricset not found")
	assert.NotNil(t, f, "factory fuction is nil")
}

func TestGetModule(t *testing.T) {
	registry := Register{
		Modulers: make(map[string]func() Moduler),
	}
	registry.Modulers[moduleName] = func() Moduler { return nil }

	config, _ := common.NewConfigFrom(ModuleConfig{
		Module: moduleName,
	})
	module, err := registry.GetModule(config)
	if assert.NoError(t, err) {
		assert.NotNil(t, module)
	}
}

func TestGetModuleInvalid(t *testing.T) {
	config, _ := common.NewConfigFrom(ModuleConfig{
		Module: moduleName,
	})

	registry := Register{}
	module, err := registry.GetModule(config)
	if assert.Error(t, err) {
		assert.Nil(t, module)
	}
}

func TestGetMetricSet(t *testing.T) {
	registry := &Register{}
	factory := func() MetricSeter { return nil }
	err := registry.AddMetricSeter(moduleName, metricSetName, factory)
	if err != nil {
		t.Fatal(err)
	}

	ms, err := registry.GetMetricSet(&Module{name: moduleName}, metricSetName)
	if assert.NoError(t, err) {
		assert.NotNil(t, ms)
	}
}
