// +build !integration

package node_stats

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	s "github.com/elastic/beats/libbeat/common/schema"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/node_stats.*.json")
	assert.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		errors := eventsMapping(reporter, input)
		for _, errs := range errors {
			if e, ok := errs.(*s.Errors); ok {
				assert.False(t, e.HasRequiredErrors(), "mapping error: %s", e)
			}
		}
		assert.True(t, len(reporter.GetEvents()) >= 1)
		assert.Equal(t, 0, len(reporter.GetErrors()))
	}
}
