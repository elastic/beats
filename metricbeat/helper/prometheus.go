package helper

import (
	"fmt"
	"io"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/elastic/beats/metricbeat/mb"
)

// Prometheus helper retrieves prometheus formatted metrics
type Prometheus struct {
	HTTP
}

// NewPrometheusClient creates new prometheus helper
func NewPrometheusClient(base mb.BaseMetricSet) (*Prometheus, error) {
	http, err := NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &Prometheus{*http}, nil
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
