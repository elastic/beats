// +build !integration

package beater_test

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	metricbeat "github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/stretchr/testify/assert"
)

const (
	moduleName    = "fake"
	metricSetName = "status"
)

// fakeMetricSet

func init() {
	if err := mb.Registry.AddMetricSet(moduleName, metricSetName, newFakeMetricSet); err != nil {
		panic(err)
	}
}

type fakeMetricSet struct {
	mb.BaseMetricSet
}

func (ms *fakeMetricSet) Fetch() (common.MapStr, error) {
	t, _ := time.Parse(time.RFC3339, "2016-05-10T23:27:58.485Z")
	return common.MapStr{"@timestamp": common.Time(t), "metric": 1}, nil
}

func newFakeMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &fakeMetricSet{BaseMetricSet: base}, nil
}

// test utilities

func newTestRegistry(t testing.TB) *mb.Register {
	r := mb.NewRegister()

	if err := r.AddMetricSet(moduleName, metricSetName, newFakeMetricSet); err != nil {
		t.Fatal(err)
	}

	return r
}

func newConfig(t testing.TB, moduleConfig interface{}) *common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

// test cases

func TestModuleWrapper(t *testing.T) {
	hosts := []string{"alpha", "beta"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
		"hosts":      hosts,
	})

	module, err := metricbeat.NewModuleWrapper(c, newTestRegistry(t))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	output := module.Start(done)

	<-output
	<-output
	close(done)

	// Validate that the channel is closed after receiving the two
	// initial events.
	select {
	case _, ok := <-output:
		if !ok {
			// Channel is closed.
			return
		} else {
			assert.Fail(t, "received unexpected event")
		}
	}
}
