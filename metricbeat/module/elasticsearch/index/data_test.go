// +build !integration

package index

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

func TestMapper(t *testing.T) {
	elasticsearch.TestMapperWithInfo(t, "../index/_meta/test/stats.*.json", eventsMapping)
}

func TestEmpty(t *testing.T) {
	input, err := ioutil.ReadFile("./_meta/test/empty.512.json")
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventsMapping(reporter, info, input)
	assert.Equal(t, 0, len(reporter.GetEvents()))
}
