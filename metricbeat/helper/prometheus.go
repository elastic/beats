package helper

import (
	"fmt"

	"github.com/elastic/beats/metricbeat/mb"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// Prometheus helper retrieves prometheus formatted metrics
type Prometheus struct {
	HTTP
}

// NewPrometheusClient creates new prometheus helper
func NewPrometheusClient(base mb.BaseMetricSet) *Prometheus {
	http := NewHTTP(base)
	return &Prometheus{*http}
}

// GetFamilies requests metric families from prometheus endpoint and returns them
func (p *Prometheus) GetFamilies() ([]*dto.MetricFamily, error) {
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
	for err == nil {
		mf := &dto.MetricFamily{}
		err = decoder.Decode(mf)
		if err == nil {
			families = append(families, mf)
		}
	}

	return families, nil
}
