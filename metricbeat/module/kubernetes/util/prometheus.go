package util

import (
	"fmt"

	"github.com/elastic/beats/metricbeat/helper"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func GetFamilies(http *helper.HTTP) ([]*dto.MetricFamily, error) {
	resp, err := http.FetchResponse()
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

func GetLabel(m *dto.Metric, label string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == label {
			return l.GetValue()
		}
	}
	return ""
}
