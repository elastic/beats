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

package collector

import (
	"regexp"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"

	"github.com/elastic/beats/libbeat/common"
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "metrics_path",
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "collector", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching prometheus data
type MetricSet struct {
	mb.BaseMetricSet
	prometheus     p.Prometheus
	includeMetrics []*regexp.Regexp
	excludeMetrics  []*regexp.Regexp
}

// New creates a new metricset
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
	}
	ms.excludeMetrics, err = compilePatternList(config.MetricsFilters.ExcludeMetrics)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to compile exclude patterns")
	}
	ms.includeMetrics, err = compilePatternList(config.MetricsFilters.IncludeMetrics)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to compile include patterns")
	}

	return ms, nil
}

// Fetch fetches data and reports it
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	families, err := m.prometheus.GetFamilies()

	eventList := map[string]common.MapStr{}
	if err != nil {
		m.addUpEvent(eventList, 0)
		for _, evt := range eventList {
			reporter.Event(mb.Event{
				RootFields: common.MapStr{"prometheus": evt},
			})
		}
		return errors.Wrap(err, "unable to decode response from prometheus endpoint")
	}

	for _, family := range families {
		if m.skipFamily(family){
			continue
		}
		promEvents := getPromEventsFromMetricFamily(family)

		for _, promEvent := range promEvents {
			labelsHash := promEvent.LabelsHash()
			if _, ok := eventList[labelsHash]; !ok {
				eventList[labelsHash] = common.MapStr{
					"metrics": common.MapStr{},
				}

				// Add default instance label if not already there
				if exists, _ := promEvent.labels.HasKey("instance"); !exists {
					promEvent.labels.Put("instance", m.Host())
				}
				// Add default job label if not already there
				if exists, _ := promEvent.labels.HasKey("job"); !exists {
					promEvent.labels.Put("job", m.Module().Name())
				}
				// Add labels
				if len(promEvent.labels) > 0 {
					eventList[labelsHash]["labels"] = promEvent.labels
				}
			}

			// Not checking anything here because we create these maps some lines before
			metrics := eventList[labelsHash]["metrics"].(common.MapStr)
			metrics.Update(promEvent.data)
		}
	}

	m.addUpEvent(eventList, 1)

	// Converts hash list to slice
	for _, e := range eventList {
		isOpen := reporter.Event(mb.Event{
			RootFields: common.MapStr{"prometheus": e},
		})
		if !isOpen {
			break
		}
	}

	return nil
}

func (m *MetricSet) addUpEvent(eventList map[string]common.MapStr, up int) {
	upPromEvent := PromEvent{
		labels: common.MapStr{
			"instance": m.Host(),
			"job":      "prometheus",
		},
	}
	eventList[upPromEvent.LabelsHash()] = common.MapStr{
		"metrics": common.MapStr{
			"up": up,
		},
		"labels": upPromEvent.labels,
	}

}

func (m *MetricSet) skipFamily(family *dto.MetricFamily) bool {
	// example:
	//	include_metrics:
	//		- node_*
	//	exclude_metrics:
	//		- node_disk_*
	//
	// This would mean that we want to keep only the metrics that start with node_ prefix but
	// are not related to disk so we exclude node_disk_* metrics from them.

	if family == nil {
		return true
	}

	// if include_metrics are defined, check if this metric should be included
	if len(m.includeMetrics) > 0  {
		if !matchMetricFamily(*family.Name, m.includeMetrics) {
			return true
		}
	}
	// now exclude the metric if it matches any of the given patterns
	if len(m.excludeMetrics) > 0 {
		if matchMetricFamily(*family.Name, m.excludeMetrics) {
			return true
		}
	}
	return false
}

func compilePatternList(patterns *[]string) ([]*regexp.Regexp, error){
	var compiledPatterns []*regexp.Regexp
	compiledPatterns = []*regexp.Regexp{}
	if patterns != nil {
		for _, pattern := range *patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				return nil, errors.Wrapf(err, "compiling pattern '%s'", pattern)
			}
			compiledPatterns = append(compiledPatterns, r)
		}
		return compiledPatterns, nil
	}
	return []*regexp.Regexp{}, nil
}

func matchMetricFamily(family string, matchMetrics []*regexp.Regexp) bool {
	for _, checkMetric := range matchMetrics {
		matched := checkMetric.MatchString(family)
		if matched {
			return true
		}
	}
	return false
}
