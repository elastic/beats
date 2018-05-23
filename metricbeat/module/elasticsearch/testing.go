// +build !integration

package elasticsearch

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

// TestMapper tests mapping methods
func TestMapper(t *testing.T, glob string, mapper func(mb.ReporterV2, []byte) []error) {
	files, err := filepath.Glob(glob)
	assert.NoError(t, err)
	// Makes sure glob matches at least 1 file
	assert.True(t, len(files) > 0)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := ioutil.ReadFile(f)
			assert.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			errors := mapper(reporter, input)
			for _, errs := range errors {
				if e, ok := errs.(*s.Errors); ok {
					assert.False(t, e.HasRequiredErrors(), "mapping error: %s", e)
				}
			}
			assert.True(t, len(reporter.GetEvents()) >= 1)
			assert.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

// TestMapperWithInfo tests mapping methods with Info fields
func TestMapperWithInfo(t *testing.T, glob string, mapper func(mb.ReporterV2, Info, []byte) []error) {
	files, err := filepath.Glob(glob)
	assert.NoError(t, err)
	// Makes sure glob matches at least 1 file
	assert.True(t, len(files) > 0)

	info := Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := ioutil.ReadFile(f)
			assert.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			errors := mapper(reporter, info, input)
			for _, errs := range errors {
				if e, ok := errs.(*s.Errors); ok {
					assert.False(t, e.HasRequiredErrors(), "mapping error: %s", e)
				}
			}
			assert.True(t, len(reporter.GetEvents()) >= 1)
			assert.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}
