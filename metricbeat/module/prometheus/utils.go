package prometheus

import (
	"fmt"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func GetMetricFamiliesFromResponse(resp *http.Response) ([]*dto.MetricFamily, error) {
	format := expfmt.ResponseFormat(resp.Header)
	if format == "" {
		return nil, fmt.Errorf("Invalid format for response of response")
	}

	decoder := expfmt.NewDecoder(resp.Body, format)
	if decoder == nil {
		return nil, fmt.Errorf("Unable to create decoder to decode response")
	}

	var err error
	result := []*dto.MetricFamily{}
	for err == nil {
		mf := &dto.MetricFamily{}
		err = decoder.Decode(mf)
		if err == nil {
			result = append(result, mf)
		}
	}

	return result, nil
}
