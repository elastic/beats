package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testMonCfgMap = MapStr{
	"id":                "testId",
	"type":              "test",
	"urls":              []string{"https://google.com"},
	"schedule":          "@every 10m",
	"service_locations": []string{"us-east"},
}

func TestNewRunnerFactoryCreatingMonitor(t *testing.T) {
	f := NewRunnerFactory()

	_, rawConfig := MockMonitorConfig(t, testMonCfgMap)
	r, err := f.Create(nil, rawConfig)
	if err != nil {
		t.Error(err)
	}
	var isUpdated bool
   go func() {
	   isUpdated = <- f.Update
   }()
	r.Start()

	monitors := f.GetMonitorsById()

	assert.Equal(t, isUpdated, true)

	for id, _ := range monitors {
		assert.Equal(t, id, "testId")
	}
	assert.Equal(t, 1, len(monitors))

}

func TestNewRunnerFactoryDeletingMonitor(t *testing.T) {
	f := NewRunnerFactory()

	_, rawConfig := MockMonitorConfig(t, testMonCfgMap)
	r, err := f.Create(nil, rawConfig)
	if err != nil {
		t.Error(err)
	}
	var isUpdated bool
	go func() {
		isUpdated = <- f.Update
	}()

	r.Start()
	monitors := f.GetMonitorsById()
	assert.Equal(t, 1, len(monitors))

	for id, _ := range monitors {
		assert.Equal(t, id, "testId")
	}

	assert.Equal(t, isUpdated, true)
	isUpdated = false
	go func() {
		isUpdated = <- f.Update
	}()
	r.Stop()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, isUpdated, true)

	monitors = f.GetMonitorsById()
	assert.Equal(t, 0, len(monitors))
}
