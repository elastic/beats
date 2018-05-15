package shard

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/routing_table.*.json")
	assert.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		eventsMapping(reporter, input)

		assert.True(t, len(reporter.GetEvents()) >= 1)
		assert.Equal(t, 0, len(reporter.GetErrors()))
	}
}
