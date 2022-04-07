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

package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"

	"github.com/pkg/errors"
)

const (
	defaultScheme = "http"
	defaultPath   = "/_stats"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

type VersionStrategy interface {
	MapEvent(info *CommonInfo, byt []byte) (mb.Event, error)
}

func init() {
	mb.Registry.MustAddMetricSet("couchdb", "server", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	http    *helper.HTTP
	fetcher VersionStrategy
	info    *CommonInfo
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet: base,
		http:          http,
		fetcher:       nil,
	}
	if err = m.retrieveFetcher(); err != nil {
		m.Logger().Warnf("error trying to get CouchDB version: '%s'. Retrying on next fetch...", err.Error())
	}

	return m, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := m.http.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error in http fetch")
	}

	if err = m.retrieveFetcher(); err != nil {
		return errors.Wrapf(err, "error trying to get CouchDB version. Retrying on next fetch...")
	}
	event, err := m.fetcher.MapEvent(m.info, content)
	if err != nil {
		return errors.Wrap(err, "error trying to get couchdb data")
	}
	reporter.Event(event)

	return nil
}

func (m *MetricSet) retrieveFetcher() (err error) {
	if m.fetcher != nil {
		return nil
	}

	m.info, err = m.getInfoFromCouchdbHost(m.Host())
	if err != nil {
		return errors.Wrap(err, "cannot start CouchDB metricbeat module")
	}

	version, err := common.NewVersion(m.info.Version)
	if err != nil {
		return errors.Wrap(err, "could not capture couchdb version")
	}

	m.Logger().Debugf("found couchdb version %d", version.Major)

	switch version.Major {
	case 1:
		m.fetcher = &V1{}
	case 2:
		m.fetcher = &V2{}
	default:
		m.fetcher = nil
	}

	return
}

// CommonInfo defines the data you receive when you make a GET request to the root path of a Couchdb server
type CommonInfo struct {
	Version string `json:"version"`
	UUID    string `json:"uuid"`
}

// Extract basic information from "/" path in Couchdb host
func (m *MetricSet) getInfoFromCouchdbHost(h string) (*CommonInfo, error) {
	c := http.DefaultClient
	c.Timeout = 30 * time.Second

	hpb := parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   "/",
	}

	hostdata, err := hpb.Build()(m.Module(), h)
	if err != nil {
		return nil, errors.Wrap(err, "error using host parser")
	}

	res, err := c.Get(hostdata.URI)
	if err != nil {
		return nil, errors.Wrap(err, "error trying to do GET request to couchdb")
	}
	defer res.Body.Close()

	var info CommonInfo
	if err = json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, errors.Wrap(err, "error trying to parse couchdb info")
	}

	return &info, nil
}
