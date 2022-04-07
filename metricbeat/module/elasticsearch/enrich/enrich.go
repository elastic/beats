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

package enrich

import (
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/helper/elastic"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "enrich", New,
		mb.WithHostParser(elasticsearch.HostParser),
	)
}

const (
	enrichStatsPath = "/_enrich/_stats"
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
	lastLicenseMessageTimestamp time.Time
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := elasticsearch.NewMetricSet(base, enrichStatsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch gathers stats for each enrich coordinator node
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	shouldSkip, err := m.ShouldSkipFetch()
	if err != nil {
		return err
	}
	if shouldSkip {
		return nil
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.GetServiceURI())
	if err != nil {
		return err
	}

	enrichUnavailableMessage, err := m.checkEnrichAvailability(info.Version.Number)
	if err != nil {
		return errors.Wrap(err, "error determining if Enrich is available")
	}

	if enrichUnavailableMessage != "" {
		if time.Since(m.lastLicenseMessageTimestamp) > 10*time.Minute {
			m.lastLicenseMessageTimestamp = time.Now()
			m.Logger().Debug(enrichUnavailableMessage)
		}
		return nil
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	return eventsMapping(r, *info, content, m.XPackEnabled)
}

func (m *MetricSet) checkEnrichAvailability(currentElasticsearchVersion *common.Version) (message string, err error) {
	isAvailable := elastic.IsFeatureAvailable(currentElasticsearchVersion, elasticsearch.EnrichStatsAPIAvailableVersion)

	if !isAvailable {
		metricsetName := m.FullyQualifiedName()
		message = "the " + metricsetName + " is only supported with Elasticsearch >= " +
			elasticsearch.EnrichStatsAPIAvailableVersion.String() + ". " +
			"You are currently running Elasticsearch " + currentElasticsearchVersion.String() + "."
		return
	}

	return "", nil
}
