package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetricSetsList(t *testing.T) {

	metricSets := map[string]*MetricSet{}
	metricSets["test1"] = &MetricSet{}
	metricSets["test2"] = &MetricSet{}

	module := Module{
		metricSets: metricSets,
	}

	assert.Equal(t, "test1, test2", module.getMetricSetsList())

}
