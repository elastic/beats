package service

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRunnerFactoryCreatingMonitor(t *testing.T) {
	fact := NewRunnerFactory(beat.Info{
		Beat:        "heartbeat",
		IndexPrefix: "heartbeat",
		Version:     "8.0.0",
	})
	myJsonString := `{
		"id":       "testId",
		"type":     "test",
		"urls":     ["https://google.com"],
		"schedule": "@every 10m",
		"service_locations": ["us-east"]
	 }`

	_, rawConfig := MockMonitorConfig(t, myJsonString)

	runner, err := fact.Create(nil, rawConfig)
	if err != nil {
		t.Error(err)
	}

	runner.Start()

	monitors := fact.GetMonitorsById()

	for id, _ := range monitors{
		assert.Equal(t, id, "testId")
	}
		assert.Equal(t, 1, len(monitors))


}

func TestNewRunnerFactoryDeletingMonitor(t *testing.T) {
	fact := NewRunnerFactory(beat.Info{
		Beat:        "heartbeat",
		IndexPrefix: "heartbeat",
		Version:     "8.0.0",
	})
	myJsonString := `{
		"id":       "testId",
		"type":     "test",
		"urls":     ["https://google.com"],
		"schedule": "@every 10m",
		"service_locations": ["us-east"]
	 }`

	_, rawConfig := MockMonitorConfig(t, myJsonString)

	runner, err := fact.Create(nil, rawConfig)
	if err != nil {
		t.Error(err)
	}

	runner.Start()

	monitors := fact.GetMonitorsById()

	assert.Equal(t, 1, len(monitors))

	for id, _ := range monitors{
		assert.Equal(t, id, "testId")
	}

	 runner.Stop()
	if err != nil {
		t.Error(err)
	}

	monitors = fact.GetMonitorsById()

	assert.Equal(t, 0, len(monitors))
}
