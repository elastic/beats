/*

Implementing mntr call from https://zookeeper.apache.org/doc/trunk/zookeeperAdmin.html

Tested with Zookeeper 3.4.8

*/
package mntr

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/zookeeper"
)

func init() {
	if err := mb.Registry.AddMetricSet("zookeeper", "mntr", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching Apache HTTPD server status.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func (m *MetricSet) Fetch(host string) (common.MapStr, error) {

	outputReader, err := zookeeper.RunCommand("mntr", host, m.Module().Config().Timeout)
	if err != nil {
		logp.Err("Error running mntr command on %s: %v", host, err)
		return nil, err
	}
	return eventMapping(outputReader), nil
}
