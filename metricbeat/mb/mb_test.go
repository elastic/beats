// +build !integration

package mb

import (
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

// EventFetcher

type testMetricSet struct {
	BaseMetricSet
}

func (m *testMetricSet) Fetch() (common.MapStr, error) {
	return nil, nil
}

// EventsFetcher

type testMetricSetEventsFetcher struct {
	BaseMetricSet
}

func (m *testMetricSetEventsFetcher) Fetch() ([]common.MapStr, error) {
	return nil, nil
}

// ReportingFetcher

type testMetricSetReportingFetcher struct {
	BaseMetricSet
}

func (m *testMetricSetReportingFetcher) Fetch(r Reporter) {}

// PushMetricSet

type testPushMetricSet struct {
	BaseMetricSet
}

func (m *testPushMetricSet) Run(r PushReporter) {}

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
		if test.err != "" {
			if err != nil {
				assert.Contains(t, err.Error(), test.err, "testcase %d", i)
			} else {
				t.Errorf("expected error '%v' in testcase %d", test.err, i)
			}
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

// TestNewModulesDuplicateHosts verifies that an error is returned by
// NewModules if any module configuration contains duplicate hosts.
func TestNewModulesDuplicateHosts(t *testing.T) {
	r := newTestRegistry(t)

	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
		"hosts":      []string{"a", "b", "a"},
	})

	_, _, err := NewModule(c, r)
	assert.Error(t, err)
}

// TestNewModulesWithDefaultMetricSet verifies that the default MetricSet is
// instantiated when no metricsets are specified in the config.
func TestNewModulesWithDefaultMetricSet(t *testing.T) {
	r := newTestRegistry(t, DefaultMetricSet())

	c := newConfig(t, map[string]interface{}{
		"module": moduleName,
	})

	_, metricSets, err := NewModule(c, r)
	if err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, metricSets, 1) {
		assert.Equal(t, metricSetName, metricSets[0].Name())
	}
}

func TestNewModulesHostParser(t *testing.T) {
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
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{metricSetName},
			"hosts":      []string{uri},
		})

		// The URI is passed through in the Host() and HostData().URI.
		assert.Equal(t, uri, ms.Host())
		assert.Equal(t, HostData{URI: uri}, ms.HostData())
	})

	t.Run("MetricSet with HostParser", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
			"hosts":      []string{uri},
		})

		// The URI is passed through in the Host() and HostData().URI.
		assert.Equal(t, host, ms.Host())
		assert.Equal(t, HostData{URI: uri, Host: host}, ms.HostData())
	})
}

func TestNewModulesMetricSetTypes(t *testing.T) {
	r := newTestRegistry(t)

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	name := "EventFetcher"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})
		_, ok := ms.(EventFetcher)
		assert.True(t, ok, name+" not implemented")
	})

	factory = func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSetEventsFetcher{base}, nil
	}

	name = "EventsFetcher"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})
		_, ok := ms.(EventsFetcher)
		assert.True(t, ok, name+" not implemented")
	})

	factory = func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSetReportingFetcher{base}, nil
	}

	name = "ReportingFetcher"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})

		_, ok := ms.(ReportingMetricSet)
		assert.True(t, ok, name+" not implemented")
	})

	factory = func(base BaseMetricSet) (MetricSet, error) {
		return &testPushMetricSet{base}, nil
	}

	name = "Push"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})
		_, ok := ms.(PushMetricSet)
		assert.True(t, ok, name+" not implemented")
	})
}

// TestNewBaseModuleFromModuleConfigStruct tests the creation a new BaseModule.
func TestNewBaseModuleFromModuleConfigStruct(t *testing.T) {
	moduleConf := DefaultModuleConfig()
	moduleConf.Module = moduleName
	moduleConf.MetricSets = []string{metricSetName}

	c := newConfig(t, moduleConf)

	baseModule, err := newBaseModuleFromConfig(c)
	assert.NoError(t, err)

	assert.Equal(t, moduleName, baseModule.Name())
	assert.Equal(t, moduleName, baseModule.Config().Module)
	assert.Equal(t, true, baseModule.Config().Enabled)
	assert.Equal(t, time.Second*10, baseModule.Config().Period)
	assert.Equal(t, time.Second*10, baseModule.Config().Timeout)
	assert.Empty(t, baseModule.Config().Hosts)
}

func newTestRegistry(t testing.TB, metricSetOptions ...MetricSetOption) *Register {
	r := NewRegister()

	if err := r.AddModule(moduleName, DefaultModuleFactory); err != nil {
		t.Fatal(err)
	}

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	if err := r.addMetricSet(moduleName, metricSetName, factory, metricSetOptions...); err != nil {
		t.Fatal(err)
	}

	return r
}

func newTestMetricSet(t testing.TB, r *Register, config map[string]interface{}) MetricSet {
	_, metricsets, err := NewModule(newConfig(t, config), r)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Len(t, metricsets, 1) {
		assert.FailNow(t, "invalid number of metricsets")
	}

	return metricsets[0]
}

func newConfig(t testing.TB, moduleConfig interface{}) *common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}
	return config
}
