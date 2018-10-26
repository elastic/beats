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

package ccr

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "ccr", New,
		mb.WithHostParser(elasticsearch.HostParser),
	)
}

const (
	ccrStatsPath = "/_ccr/stats"
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("the " + base.FullyQualifiedName() + " metricset is beta")

	ms, err := elasticsearch.NewMetricSet(base, ccrStatsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch gathers stats for each follower shard from the _ccr/stats API
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.HostData().SanitizedURI+ccrStatsPath)
	if err != nil {
		err = errors.Wrap(err, "error determining if connected Elasticsearch node is master")
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	// Not master, no event sent
	if !isMaster {
		m.Log.Debug("trying to fetch ccr stats from a non-master node")
		return
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.HostData().SanitizedURI+ccrStatsPath)
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	elasticsearchVersion := info.Version.Number
	isCCRStatsAPIAvailable, err := elasticsearch.IsCCRStatsAPIAvailable(elasticsearchVersion)
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	if !isCCRStatsAPIAvailable {
		const errorMsg = "the %v metricset is only supported with Elasticsearch >= %v. " +
			"You are currently running Elasticsearch %v"
		err = fmt.Errorf(errorMsg, m.FullyQualifiedName(), elasticsearch.CCRStatsAPIAvailableVersion, elasticsearchVersion)
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	if m.XPack {
		err = eventsMappingXPack(r, m, *info, content)
	} else {
		err = eventsMapping(r, *info, content)
	}

	if err != nil {
		m.Log.Error(err)
		return
	}
}
