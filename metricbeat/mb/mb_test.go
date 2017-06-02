// +build !integration

package mb

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

type testModule struct {
	BaseModule
	hostParser func(string) (HostData, error)
}

func (m testModule) ParseHost(host string) (HostData, error) {
	return m.hostParser(host)
}

type testMetricSet struct {
	BaseMetricSet
}

func (m *testMetricSet) Fetch() (common.MapStr, error) {
	return nil, nil
}

func TestModuleConfig(t *testing.T) {
	tests := []struct {
		in  interface{}
		out ModuleConfig
		err string
	}{
		{
			in:  map[string]interface{}{},
			err: "missing required field accessing 'module'",
		},
		{
			in: map[string]interface{}{
				"module": "example",
			},
			err: "missing required field accessing 'metricsets'",
		},
		{
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{},
			},
			err: "empty field accessing 'metricsets'",
		},
		{
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
			},
			out: ModuleConfig{
				Module:     "example",
				MetricSets: []string{"test"},
				Enabled:    true,
				Period:     time.Second * 10,
				Timeout:    0,
			},
		},
		{
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
				"period":     -1,
			},
			err: "negative value accessing 'period'",
		},
		{
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
				"timeout":    -1,
			},
			err: "negative value accessing 'timeout'",
		},
	}

	for i, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		unpackedConfig := DefaultModuleConfig()
		err = c.Unpack(&unpackedConfig)
		if err != nil && test.err == "" {
			t.Errorf("unexpected error while unpacking in testcase %d: %v", i, err)
			continue
		}
		if test.err != "" &&
			assert.Error(t, err, fmt.Sprintf("expected '%v' in testcase %d", test.err, i)) {
			assert.Contains(t, err.Error(), test.err, "testcase %d", i)
			continue
		}

		assert.Equal(t, test.out, unpackedConfig)
	}
}

// TestModuleConfigDefaults validates that the default values are not changed.
// Any changes to this test case are probably indicators of non-backwards
// compatible changes affect all modules (including community modules).
func TestModuleConfigDefaults(t *testing.T) {
	c, err := common.NewConfigFrom(map[string]interface{}{
		"module":     "mymodule",
		"metricsets": []string{"mymetricset"},
	})
	if err != nil {
		t.Fatal(err)
	}

	mc := DefaultModuleConfig()
	err = c.Unpack(&mc)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, mc.Enabled)
	assert.Equal(t, time.Second*10, mc.Period)
	assert.Equal(t, time.Second*0, mc.Timeout)
	assert.Empty(t, mc.Hosts)
}

// TestNewModulesWithEmptyModulesConfig verifies that an error is returned if
// the modules configuration list is empty.
func TestNewModulesWithEmptyModulesConfig(t *testing.T) {
	r := newTestRegistry(t)
	_, err := NewModules(nil, r)
	assert.Equal(t, ErrEmptyConfig, err)
}

// TestNewModulesWithAllDisabled verifies that an error is returned if all
// modules defined in the config are disabled.
func TestNewModulesWithAllDisabled(t *testing.T) {
	r := newTestRegistry(t)

	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
		"enabled":    false,
	})

	_, err := NewModules(c, r)
	assert.Equal(t, ErrAllModulesDisabled, err)
}

// TestNewModulesDuplicateHosts verifies that an error is returned by
// NewModules if any module configuration contains duplicate hosts.
func TestNewModulesDuplicateHosts(t *testing.T) {
	r := newTestRegistry(t)

	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
		"hosts":      []string{"a", "b", "a"},
	})

	_, err := NewModules(c, r)
	assert.Error(t, err)
}

func TestNewModules(t *testing.T) {
	const (
		name = "HostParser"
		host = "example.com"
		uri  = "http://" + host
	)

	r := newTestRegistry(t)

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	hostParser := func(m Module, rawHost string) (HostData, error) {
		return HostData{URI: uri, Host: host}, nil
	}

	if err := r.AddMetricSet(moduleName, name, factory, hostParser); err != nil {
		t.Fatal(err)
	}

	t.Run("MetricSet without HostParser", func(t *testing.T) {
		c := newConfig(t, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{metricSetName},
			"hosts":      []string{uri},
		})

		modules, err := NewModules(c, r)
		if err != nil {
			t.Fatal(err)
		}

		for _, metricSets := range modules {
			metricSet := metricSets[0]

			// The URI is passed through in the Host() and HostData().URI.
			assert.Equal(t, uri, metricSet.Host())
			assert.Equal(t, HostData{URI: uri}, metricSet.HostData())
			return
		}
		assert.FailNow(t, "no modules found")
	})

	t.Run("MetricSet with HostParser", func(t *testing.T) {
		c := newConfig(t, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
			"hosts":      []string{uri},
		})

		modules, err := NewModules(c, r)
		if err != nil {
			t.Fatal(err)
		}

		for _, metricSets := range modules {
			metricSet := metricSets[0]

			// The URI is passed through in the Host() and HostData().URI.
			assert.Equal(t, host, metricSet.Host())
			assert.Equal(t, HostData{URI: uri, Host: host}, metricSet.HostData())
			return
		}
		assert.FailNow(t, "no modules found")
	})
}

// TestNewBaseModuleFromModuleConfigStruct tests the creation a new BaseModule.
func TestNewBaseModuleFromModuleConfigStruct(t *testing.T) {
	moduleConf := DefaultModuleConfig()
	moduleConf.Module = moduleName
	moduleConf.MetricSets = []string{metricSetName}

	c := newConfig(t, moduleConf)

	baseModule, err := newBaseModuleFromConfig(c[0])
	assert.NoError(t, err)

	assert.Equal(t, moduleName, baseModule.Name())
	assert.Equal(t, moduleName, baseModule.Config().Module)
	assert.Equal(t, true, baseModule.Config().Enabled)
	assert.Equal(t, time.Second*10, baseModule.Config().Period)
	assert.Equal(t, time.Second*10, baseModule.Config().Timeout)
	assert.Empty(t, baseModule.Config().Hosts)
}

func newTestRegistry(t testing.TB) *Register {
	r := NewRegister()

	if err := r.AddModule(moduleName, DefaultModuleFactory); err != nil {
		t.Fatal(err)
	}

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	if err := r.AddMetricSet(moduleName, metricSetName, factory); err != nil {
		t.Fatal(err)
	}

	return r
}

func newConfig(t testing.TB, moduleConfig interface{}) []*common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}

	return []*common.Config{config}
}
