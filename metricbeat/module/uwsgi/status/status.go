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

package status

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/uwsgi"
)

func init() {
	mb.Registry.MustAddMetricSet("uwsgi", "status", New,
		mb.WithHostParser(uwsgi.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching uwsgi metrics from StatServer.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{BaseMetricSet: base}, nil
}

func fetchStatData(URL string) ([]byte, error) {
	var reader io.Reader

	u, err := url.Parse(URL)
	if err != nil {

		return nil, errors.Wrap(err, "parsing uwsgi stats url failed")
	}

	switch u.Scheme {
	case "tcp":
		conn, err := net.Dial(u.Scheme, u.Host)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		reader = conn
	case "unix":
		path := strings.Replace(URL, "unix://", "", -1)
		conn, err := net.Dial(u.Scheme, path)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		reader = conn
	case "http", "https":
		res, err := http.Get(u.String())
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {

			return nil, fmt.Errorf("failed to fetch uwsgi status with code: %d", res.StatusCode)
		}
		reader = res.Body
	default:
		return nil, errors.New("unknown scheme")
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "uwsgi data read failed")
	}

	return data, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := fetchStatData(m.HostData().URI)
	if err != nil {
		reporter.Event(mb.Event{MetricSetFields: common.MapStr{
			"error": err.Error(),
		}},
		)
		return err
	}
	return eventsMapping(content, reporter)
}
