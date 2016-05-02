// +build !integration

package helper

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

// TestMetricSeterState tests if a metricset persists its state during multiple Fetch requests
func TestMetricSeterState(t *testing.T) {
	module := &Module{}

	metricSet, err := NewMetricSet("mockmetricset", NewMockMetricSeter, module)
	assert.NoError(t, err)

	event, _ := metricSet.MetricSeter.Fetch(metricSet, "")
	assert.Equal(t, 1, event["counter"])

	event, _ = metricSet.MetricSeter.Fetch(metricSet, "")
	assert.Equal(t, 2, event["counter"])
}

// TestMetricSetTwoInstances makes sure that in case of two different MetricSet instance, MetricSeter don't share state
func TestMetricSetTwoInstances(t *testing.T) {
	module := &Module{}

	metricSet1, err1 := NewMetricSet("mockmetricset1", NewMockMetricSeter, module)
	metricSet2, err2 := NewMetricSet("mockmetricset2", NewMockMetricSeter, module)
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	event, _ := metricSet1.MetricSeter.Fetch(metricSet1, "")
	assert.Equal(t, 1, event["counter"])

	event, _ = metricSet2.MetricSeter.Fetch(metricSet2, "")
	assert.Equal(t, 1, event["counter"])
}

func TestCreateEvent(t *testing.T) {
	module := &Module{}
	metricSet, _ := NewMetricSet("mockmetricset1", NewMockMetricSeter, module)

	event := common.MapStr{}
	host := "localhost"
	rtt, _ := time.ParseDuration("1s")
	event = metricSet.createEvent(event, host, rtt, nil)

	assert.Equal(t, host, event["metricset-host"])

	_, ok := event["error"]
	assert.False(t, ok)
	assert.Equal(t, rtt.Nanoseconds()/1000, event["rtt"])
}

func TestCreateEventError(t *testing.T) {
	module := &Module{}
	metricSet, _ := NewMetricSet("mockmetricset1", NewMockMetricSeter, module)

	event := common.MapStr{}
	host := "localhost"
	rtt, _ := time.ParseDuration("1s")

	eventErr := fmt.Errorf("hello world")
	event = metricSet.createEvent(event, host, rtt, eventErr)

	assert.Equal(t, host, event["metricset-host"])
	assert.Equal(t, eventErr.Error(), event["error"])
}

func TestCreateEventNoHost(t *testing.T) {
	module := &Module{}
	metricSet, _ := NewMetricSet("mockmetricset1", NewMockMetricSeter, module)

	event := common.MapStr{}
	rtt, _ := time.ParseDuration("1s")

	event = metricSet.createEvent(event, "", rtt, nil)

	_, ok := event["metricset-host"]
	assert.False(t, ok)
}

func TestFetcherCounter(t *testing.T) {
	module := &Module{}
	m, _ := NewMetricSet("mockmetricset1", NewMockMetricSeter, module)

	counter := m.incrementFetcher()
	assert.Equal(t, uint32(1), counter)

	counter = m.incrementFetcher()
	assert.Equal(t, uint32(2), counter)

	counter = m.decrementFetcher()
	assert.Equal(t, uint32(1), counter)
}

/*** Mock tests objects ***/

// New creates new instance of MetricSeter
func NewMockMetricSeter() MetricSeter {
	return &MockMetricSeter{
		counter: 0,
	}
}

type MockMetricSeter struct {
	counter int
}

func (m *MockMetricSeter) Setup(ms *MetricSet) error {
	return nil
}

func (m *MockMetricSeter) Fetch(ms *MetricSet, host string) (event common.MapStr, err error) {
	m.counter += 1

	event = common.MapStr{
		"counter": m.counter,
	}

	return event, nil
}
