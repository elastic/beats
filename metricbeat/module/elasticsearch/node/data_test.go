// +build !integration

package node

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	s "github.com/elastic/beats/libbeat/common/schema"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMappings(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/node.*.json")
	assert.NoError(t, err)

	for _, f := range files {
		content, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		errors := eventsMapping(reporter, content)
		for _, errs := range errors {
			if e, ok := errs.(*s.Errors); ok {
				assert.False(t, e.HasRequiredErrors(), "mapping error: %s", e)
			}
		}
		assert.True(t, len(reporter.GetEvents()) >= 1)
		assert.Equal(t, 0, len(reporter.GetErrors()))
	}
}

func TestInvalid(t *testing.T) {
	file := "./_meta/test/invalid.json"

	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	errs := eventsMapping(reporter, content)

	errors, ok := errs[0].(*s.Errors)
	if ok {
		assert.True(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
	} else {
		t.Error(err)
	}
}
