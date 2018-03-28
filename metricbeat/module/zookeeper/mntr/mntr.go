/*
Package mntr fetches metrics from ZooKeeper by using the mntr command which was
added to ZooKeeper in version 3.4.0.

See the mntr command documentation at
https://zookeeper.apache.org/doc/trunk/zookeeperAdmin.html

ZooKeeper mntr Command Output

  $ echo mntr | nc localhost 2185
  zk_version	3.4.8--1, built on 02/06/2016 03:18 GMT
  zk_avg_latency	0
  zk_max_latency	0
  zk_min_latency	0
  zk_packets_received	10
  zk_packets_sent	9
  zk_num_alive_connections	1
  zk_outstanding_requests	0
  zk_server_state	standalone
  zk_znode_count	4
  zk_watch_count	0
  zk_ephemerals_count	0
  zk_approximate_data_size	27
  zk_open_file_descriptor_count	25
  zk_max_file_descriptor_count	1048576
*/
package mntr

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/zookeeper"

	"github.com/pkg/errors"
)

func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "mntr", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching ZooKeeper health metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches metrics from ZooKeeper by making a tcp connection to the
// command port and sending the "mntr" command and parsing the output.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	outputReader, err := zookeeper.RunCommand("mntr", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "mntr command failed")
	}
	return eventMapping(outputReader), nil
}
