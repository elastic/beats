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
		in  map[string]interface{}
		out ModuleConfig
		err string
	}{
		{
			in:  map[string]interface{}{},
			out: defaultModuleConfig,
		},
	}

	for _, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		mc := defaultModuleConfig
		err = c.Unpack(&mc)
		if test.err != "" {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.err)
			continue
		}

		assert.Equal(t, test.out, mc)
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

	mc := defaultModuleConfig
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

func newConfig(t testing.TB, moduleConfig map[string]interface{}) []*common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}

	return []*common.Config{config}
}
