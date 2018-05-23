// +build !integration

package node

import (
	"io/ioutil"
	"testing"

	s "github.com/elastic/beats/libbeat/common/schema"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func TestGetMappings(t *testing.T) {
	elasticsearch.TestMapper(t, "./_meta/test/node.*.json", eventsMapping)
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
