// +build !integration

package mb

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

type testMetricSet struct {
	BaseMetricSet
}

func (m *testMetricSet) Fetch(host string) (common.MapStr, error) {
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
				Period:     time.Second,
				Timeout:    time.Second,
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
			assert.Error(t, err, "expected '%v' in testcase %d", test.err, i) {
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
	assert.Equal(t, time.Second, mc.Period)
	assert.Equal(t, time.Second, mc.Timeout)
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

// TestNewBaseModuleFromModuleConfigStruct tests using a ModuleConfig struct
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
	assert.Equal(t, time.Second, baseModule.Config().Period)
	assert.Equal(t, time.Second, baseModule.Config().Timeout)
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
