// +build !integration

package node_stats

import (
	"testing"

	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func TestStats(t *testing.T) {
	elasticsearch.TestMapper(t, "./_meta/test/node_stats.*.json", eventsMapping)
}
