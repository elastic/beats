package prometheus

import (
	"fmt"
	"io"
	"net/http"

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
		} else {
			families = append(families, mf)
		}
	}

	return families, nil
}

// MetricsMapping defines mapping settings for Prometheus metrics, to be used with `GetProcessedMetrics`
type MetricsMapping struct {
	// Metrics translates from from prometheus metric name to Metricbeat fields
	Metrics map[string]MetricMap

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

			// Convert labels
			labels := common.MapStr{}
			keyLabels := common.MapStr{}
			for k, v := range getLabels(metric) {
				if l, ok := mapping.Labels[k]; ok {
					if l.IsKey() {
						keyLabels[l.GetField()] = v
					} else {
						labels[l.GetField()] = v
					}
				}
			}

			event := getEvent(eventsMap, keyLabels)
			// Empty field means we ignore the metric but still process its labels
			if field != "" {
				event[field] = value
			}

			event.Update(labels)
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
	return events, nil
}

func getEvent(m map[string]common.MapStr, labels common.MapStr) common.MapStr {
	hash := labels.String()
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
			labels[label.GetName()] = label.GetValue()
		}
	}
	return labels
}
