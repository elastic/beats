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

package openmetrics

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const acceptHeader = `application/openmetrics-text; version=1.0.0; charset=utf-8,text/plain`

// OpenMetrics helper retrieves openmetrics formatted metrics
// This interface needs to use TextParse
type OpenMetrics interface {
	// GetFamilies requests metric families from openmetrics endpoint and returns them
	GetFamilies() ([]*prometheus.MetricFamily, error)

	GetProcessedMetrics(mapping *MetricsMapping) ([]mapstr.M, error)

	ProcessMetrics(families []*prometheus.MetricFamily, mapping *MetricsMapping) ([]mapstr.M, error)

	ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error
}

type openmetrics struct {
	httpfetcher
	logger *logp.Logger
}

type httpfetcher interface {
	FetchResponse() (*http.Response, error)
}

// NewOpenMetricsClient creates new openmetrics helper
func NewOpenMetricsClient(base mb.BaseMetricSet) (OpenMetrics, error) {
	httpclient, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	httpclient.SetHeaderDefault("Accept", acceptHeader)
	httpclient.SetHeaderDefault("Accept-Encoding", "gzip")
	return &openmetrics{httpclient, base.Logger()}, nil
}

// GetFamilies requests metric families from openmetrics endpoint and returns them
func (p *openmetrics) GetFamilies() ([]*prometheus.MetricFamily, error) {
	var reader io.Reader

	resp, err := p.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") == "gzip" {
		greader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer greader.Close()
		reader = greader
	} else {
		reader = resp.Body
	}

	if resp.StatusCode > 399 {
		bodyBytes, err := io.ReadAll(reader)
		if err == nil {
			p.logger.Debug("error received from openmetrics endpoint: ", string(bodyBytes))
		}
		return nil, fmt.Errorf("unexpected status code %d from server", resp.StatusCode)
	}

	contentType := prometheus.GetContentType(resp.Header)
	if contentType == "" {
		return nil, fmt.Errorf("Invalid format for response of response")
	}

	appendTime := time.Now().Round(0)
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	families, err := prometheus.ParseMetricFamilies(b, contentType, appendTime, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to parse families: %w", err)
	}

	return families, nil
}

// MetricsMapping defines mapping settings for OpenMetrics metrics, to be used with `GetProcessedMetrics`
type MetricsMapping struct {
	// Metrics translates from openmetrics metric name to Metricbeat fields
	Metrics map[string]MetricMap

	// Namespace for metrics managed by this mapping
	Namespace string

	// Labels translate from openmetrics label names to Metricbeat fields
	Labels map[string]LabelMap

	// ExtraFields adds the given fields to all events coming from `GetProcessedMetrics`
	ExtraFields map[string]string
}

func (p *openmetrics) ProcessMetrics(families []*prometheus.MetricFamily, mapping *MetricsMapping) ([]mapstr.M, error) {

	eventsMap := map[string]mapstr.M{}
	infoMetrics := []*infoMetricData{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			m, ok := mapping.Metrics[family.GetName()]
			if m == nil || !ok {
				// Ignore unknown metrics
				continue
			}

			field := m.GetField()
			value := m.GetValue(metric)

			// Ignore retrieval errors (bad conf)
			if value == nil {
				continue
			}

			storeAllLabels := false
			labelsLocation := ""
			var extraFields mapstr.M
			if m != nil {
				c := m.GetConfiguration()
				storeAllLabels = c.StoreNonMappedLabels
				labelsLocation = c.NonMappedLabelsPlacement
				extraFields = c.ExtraFields
			}

			// Apply extra options
			allLabels := getLabels(metric)
			for _, option := range m.GetOptions() {
				field, value, allLabels = option.Process(field, value, allLabels)
			}

			// Convert labels
			labels := mapstr.M{}
			keyLabels := mapstr.M{}
			for k, v := range allLabels {
				if l, ok := mapping.Labels[k]; ok {
					if l.IsKey() {
						_, _ = keyLabels.Put(l.GetField(), v)
					} else {
						_, _ = labels.Put(l.GetField(), v)
					}
				} else if storeAllLabels {
					// if label for this metric is not found at the label mappings but
					// it is configured to store any labels found, make it so
					_, _ = labels.Put(labelsLocation+"."+k, v)
				}
			}

			// if extra fields have been added through metric configuration
			// add them to labels.
			//
			// not considering these extra fields to be keylabels as that case
			// have not appeared yet
			for k, v := range extraFields {
				_, _ = labels.Put(k, v)
			}

			// Keep a info document if it's an infoMetric
			if _, ok = m.(*infoMetric); ok {
				labels.DeepUpdate(keyLabels)
				infoMetrics = append(infoMetrics, &infoMetricData{
					Labels: keyLabels,
					Meta:   labels,
				})
				continue
			}

			if field != "" {
				event := getEvent(eventsMap, keyLabels)
				update := mapstr.M{}
				_, _ = update.Put(field, value)
				// value may be a mapstr (for histograms and summaries), do a deep update to avoid smashing existing fields
				event.DeepUpdate(update)

				event.DeepUpdate(labels)
			}
		}
	}

	// populate events array from values in eventsMap
	events := make([]mapstr.M, 0, len(eventsMap))
	for _, event := range eventsMap {
		// Add extra fields
		for k, v := range mapping.ExtraFields {
			event[k] = v
		}
		events = append(events, event)
	}

	// fill info from infoMetrics
	for _, info := range infoMetrics {
		for _, event := range events {
			found := true
			for k, v := range info.Labels.Flatten() {
				value, err := event.GetValue(k)
				if err != nil || v != value {
					found = false
					break
				}
			}

			// fill info from this metric
			if found {
				event.DeepUpdate(info.Meta)
			}
		}
	}

	return events, nil
}

func (p *openmetrics) GetProcessedMetrics(mapping *MetricsMapping) ([]mapstr.M, error) {
	families, err := p.GetFamilies()
	if err != nil {
		return nil, err
	}
	return p.ProcessMetrics(families, mapping)
}

// infoMetricData keeps data about an infoMetric
type infoMetricData struct {
	Labels mapstr.M
	Meta   mapstr.M
}

func (p *openmetrics) ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error {
	events, err := p.GetProcessedMetrics(mapping)
	if err != nil {
		return fmt.Errorf("error getting processed metrics: %w", err)
	}
	for _, event := range events {
		r.Event(mb.Event{
			MetricSetFields: event,
			Namespace:       mapping.Namespace,
		})
	}

	return nil
}

func getEvent(m map[string]mapstr.M, labels mapstr.M) mapstr.M {
	hash := labels.String()
	res, ok := m[hash]
	if !ok {
		res = labels
		m[hash] = res
	}
	return res
}

func getLabels(metric *prometheus.OpenMetric) mapstr.M {
	labels := mapstr.M{}
	for _, label := range metric.GetLabel() {
		if label.Name != "" && label.Value != "" {
			_, _ = labels.Put(label.Name, label.Value)
		}
	}
	return labels
}

// CompilePatternList compiles a pattern list and returns the list of the compiled patterns
func CompilePatternList(patterns *[]string) ([]*regexp.Regexp, error) {
	var compiledPatterns []*regexp.Regexp
	compiledPatterns = []*regexp.Regexp{}
	if patterns != nil {
		for _, pattern := range *patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to compile pattern '%s': %w", pattern, err)
			}
			compiledPatterns = append(compiledPatterns, r)
		}
		return compiledPatterns, nil
	}
	return []*regexp.Regexp{}, nil
}

// MatchMetricFamily checks if the given family/metric name matches any of the given patterns
func MatchMetricFamily(family string, matchMetrics []*regexp.Regexp) bool {
	for _, checkMetric := range matchMetrics {
		matched := checkMetric.MatchString(family)
		if matched {
			return true
		}
	}
	return false
}
