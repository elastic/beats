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
Package server fetches metrics from ZooKeeper by using the srvr command

See the srvr command documentation at
https://zookeeper.apache.org/doc/current/zookeeperAdmin.html

ZooKeeper srvr Command Output

  $ echo srvr | nc localhost 2181
	Zookeeper version: 3.4.13-2d71af4dbe22557fda74f9a9b4309b15a7487f03, built on 06/29/2018 04:05 GMT
Latency min/avg/max: 1/2/3
Received: 46
Sent: 45
Connections: 1
Outstanding: 0
Zxid: 0x700601132
Mode: standalone
Node count: 4
Proposal sizes last/min/max: -3/-999/-1


*/
package server

import (
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/zookeeper"
)

func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "server", New,
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
// command port and sending the "srvr" command and parsing the output.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	outputReader, err := zookeeper.RunCommand("srvr", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "srvr command failed")

	}

	metricsetFields, version, err := parseSrvr(outputReader, m.Logger())
	if err != nil {
		return errors.Wrap(err, "error parsing srvr output")
	}

	serverID, err := zookeeper.ServerID(m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "error obtaining server id")
	}

	event := mb.Event{
		MetricSetFields: metricsetFields,
		RootFields: common.MapStr{
			"service": common.MapStr{
				"node": common.MapStr{
					"name": serverID,
				},
				"version": version,
			},
		},
	}

	reporter.Event(event)
	return nil
}
