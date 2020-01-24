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

package stats

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("memcached", "stats", New,
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet

	network    string
	socketPath string
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Network    string `config:"network"`
		SocketPath string `config:"socket_path"`
	}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		network:       config.Network,
		socketPath:    config.SocketPath,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	network, address, err := m.getNetworkAndAddress()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	conn, err := net.DialTimeout(network, address, m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}
	defer conn.Close()

	_, err = conn.Write([]byte("stats\n"))
	if err != nil {
		return errors.Wrap(err, "error in connection")
	}

	scanner := bufio.NewScanner(conn)

	data := map[string]interface{}{}

	for scanner.Scan() {
		text := scanner.Text()
		if text == "END" {
			break
		}

		// Split entries which look like: STAT time 1488291730
		entries := strings.Split(text, " ")
		if len(entries) == 3 {
			data[entries[1]] = entries[2]
		}
	}

	event, _ := schema.Apply(data)

	reporter.Event(mb.Event{MetricSetFields: event})

	return nil
}

func (m *MetricSet) getNetworkAndAddress() (network string, address string, err error) {
	switch m.network {
	case "", "tcp":
		network = "tcp"
		address = m.Host()
	case "unix":
		network = "unix"
		address = m.socketPath
	default:
		err = fmt.Errorf("unsupported network: %s", m.network)
	}
	return
}
