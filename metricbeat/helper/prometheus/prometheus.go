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

package prometheus

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

// Prometheus helper retrieves prometheus formatted metrics
type Prometheus interface {
	// GetFamilies requests metric families from prometheus endpoint and returns them
	GetFamilies() ([]*dto.MetricFamily, error)

	GetProcessedMetrics(mapping *MetricsMapping) ([]common.MapStr, error)

	ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error
}

type prometheus struct {
	httpfetcher
}

type httpfetcher interface {
	FetchResponse() (*http.Response, error)
}

// NewPrometheusClient creates new prometheus helper
func NewPrometheusClient(base mb.BaseMetricSet) (Prometheus, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &prometheus{http}, nil
}

// GetFamilies requests metric families from prometheus endpoint and returns them
func (p *prometheus) GetFamilies() ([]*dto.MetricFamily, error) {
	resp, err := p.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	format := expfmt.ResponseFormat(resp.Header)
	if format == "" {
		return nil, fmt.Errorf("Invalid format for response of response")
	}

	decoder := expfmt.NewDecoder(resp.Body, format)
	if decoder == nil {
		return nil, fmt.Errorf("Unable to create decoder to decode response")
	}

	families := []*dto.MetricFamily{}
	for {
		mf := &dto.MetricFamily{}
		err = decoder.Decode(mf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "decoding of metric family failed")
		} else {
			families = append(families, mf)
		}
	}

	return families, nil
}

// MetricsMapping defines mapping settings for Prometheus metrics, to be used with `GetProcessedMetrics`
type MetricsMapping struct {
	// Metrics translates from prometheus metric name to Metricbeat fields
	Metrics map[string]MetricMap

	// Namespace for metrics managed by this mapping
	Namespace string

	// Labels translate from prometheus label names to Metricbeat fields
	Labels map[string]LabelMap

	// ExtraFields adds the given fields to all events coming from `GetProcessedMetrics`
	ExtraFields map[string]string
}

func (p *prometheus) GetProcessedMetrics(mapping *MetricsMapping) ([]common.MapStr, error) {
	families, err := p.GetFamilies()
	if err != nil {
		return nil, err
	}

	eventsMap := map[string]common.MapStr{}
	infoMetrics := []*infoMetricData{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			m, ok := mapping.Metrics[family.GetName()]

			// Ignore unknown metrics
			if !ok {
				continue
			}

			field := m.GetField()
			value := m.GetValue(metric)

			// Ignore retrieval errors (bad conf)
			if value == nil {
				continue
			}

			// Apply extra options
			allLabels := getLabels(metric)
			for _, option := range m.GetOptions() {
				field, value, allLabels = option.Process(field, value, allLabels)
			}

			// Convert labels
			labels := common.MapStr{}
			keyLabels := common.MapStr{}
			for k, v := range allLabels {
				if l, ok := mapping.Labels[k]; ok {
					if l.IsKey() {
						keyLabels.Put(l.GetField(), v)
					} else {
						labels.Put(l.GetField(), v)
					}
				}
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
				update := common.MapStr{}
				update.Put(field, value)
				// value may be a mapstr (for histograms and summaries), do a deep update to avoid smashing existing fields
				event.DeepUpdate(update)

				event.DeepUpdate(labels)
			}
		}
	}

	// populate events array from values in eventsMap
	events := make([]common.MapStr, 0, len(eventsMap))
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

// infoMetricData keeps data about an infoMetric
type infoMetricData struct {
	Labels common.MapStr
	Meta   common.MapStr
}

func (p *prometheus) ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error {
	events, err := p.GetProcessedMetrics(mapping)
	if err != nil {
		return errors.Wrap(err, "error getting processed metrics")
	}
	for _, event := range events {
		r.Event(mb.Event{
			MetricSetFields: event,
			Namespace:       mapping.Namespace,
		})
	}

	return nil
}

func getEvent(m map[string]common.MapStr, labels common.MapStr) common.MapStr {
	hash := LabelHash(labels.Flatten())
	res, ok := m[hash]
	if !ok {
		res = labels
		m[hash] = res
	}
	return res
}

func getLabels(metric *dto.Metric) common.MapStr {
	labels := common.MapStr{}
	for _, label := range metric.GetLabel() {
		if label.GetName() != "" && label.GetValue() != "" {
			labels.Put(label.GetName(), label.GetValue())
		}
	}
	return labels
}
