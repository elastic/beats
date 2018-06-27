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
	"net"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("memcached", "stats", New,
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The memcached stats metricset is beta")

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {
	conn, err := net.DialTimeout("tcp", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte("stats\n"))
	if err != nil {
		return nil, err
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
	return event, nil
}
