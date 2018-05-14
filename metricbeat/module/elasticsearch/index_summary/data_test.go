// +build !integration

package index_summary

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
	elasticsearch.TestMapperWithInfo(t, "../index/_meta/test/stats.*.json", eventMapping)
}

func TestEmpty(t *testing.T) {
	input, err := ioutil.ReadFile("../index/_meta/test/empty.512.json")
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventMapping(reporter, info, input)
	assert.Equal(t, 1, len(reporter.GetEvents()))
}
