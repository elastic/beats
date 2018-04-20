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

func TestIndex(t *testing.T) {
	versions := []string{
		"175", "201", "240", "512", "623",
	}

	_, err := ioutil.ReadFile("./_meta/test/output.json")
	assert.NoError(t, err)

	for _, v := range versions {
		input, err := ioutil.ReadFile("./_meta/test/stats." + v + ".json")
		assert.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		eventsMapping(reporter, info, input)
		assert.True(t, len(reporter.GetEvents()) >= 1)
		assert.Equal(t, 0, len(reporter.GetErrors()))
	}
}

func TestEmpty(t *testing.T) {
	input, err := ioutil.ReadFile("./_meta/test/empty.512.json")
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventsMapping(reporter, info, input)
	assert.Equal(t, 0, len(reporter.GetEvents()))
}
