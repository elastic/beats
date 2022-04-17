// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/zookeeper"

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
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	outputReader, err := zookeeper.RunCommand("mntr", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "mntr command failed")
	}

	serverID, err := zookeeper.ServerID(m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "error obtaining server id")
	}

	eventMapping(serverID, outputReader, r, m.Logger())
	return nil
}
