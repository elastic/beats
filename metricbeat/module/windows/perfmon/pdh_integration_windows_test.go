// +build integration windows

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExistingCounter(t *testing.T) {
	config := make([]CounterConfig, 1)
	config[0].Name = "process"
	config[0].Group = make([]CounterConfigGroup, 1)
	config[0].Group[0].Alias = "processor_time"
	config[0].Group[0].Query = "\\Processor Information(_Total)\\% Processor Time"
	handle, err := GetHandle(config)

	assert.Zero(t, err)

	err = CloseQuery(handle.query)

	assert.Zero(t, err)
}

func TestNonExistingCounter(t *testing.T) {
	config := make([]CounterConfig, 1)
	config[0].Name = "process"
	config[0].Group = make([]CounterConfigGroup, 1)
	config[0].Group[0].Alias = "processor_performance"
	config[0].Group[0].Query = "\\Processor Information(_Total)\\not existing counter"
	handle, err := GetHandle(config)

	assert.Equal(t, 3221228473, int(err))

	if handle != nil {
		err = CloseQuery(handle.query)

		assert.Zero(t, err)
	}
}

func TestNonExistingObject(t *testing.T) {
	config := make([]CounterConfig, 1)
	config[0].Name = "process"
	config[0].Group = make([]CounterConfigGroup, 1)
	config[0].Group[0].Alias = "processor_performance"
	config[0].Group[0].Query = "\\non existing object\\% Processor Performance"
	handle, err := GetHandle(config)

	assert.Equal(t, 3221228472, int(err))

	if handle != nil {
		err = CloseQuery(handle.query)

		assert.Zero(t, err)
	}
}
