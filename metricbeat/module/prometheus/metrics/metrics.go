package metrics

//read data from the /metrics endpoint and generate metrics from the same
import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/prometheus/common/expfmt"

	_ "github.com/elastic/beats/metricbeat/module/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func init() {
	helper.Registry.AddMetricSeter("prometheus", "metrics", MetricSeter{})
}

type MetricSeter struct{}

func (m MetricSeter) Setup() error {
	return nil
}

func (m MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {
	hosts := ms.Config.Hosts

	for _, host := range hosts {
		resp, err := http.Get(host + "metrics")
		defer resp.Body.Close()

		if err != nil {
			logp.Err("Error during Request: %s", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP Error %s: %s", resp.StatusCode, resp.Status)
		}

		format := expfmt.ResponseFormat(resp.Header)

		decoder := expfmt.NewDecoder(resp.Body, format)
		result := []*dto.MetricFamily{}
		for err == nil {
			mf := &dto.MetricFamily{}
			err = decoder.Decode(mf)
			if err == nil {
				result = append(result, mf)
			}
		}

		for _, mf := range result {
			name := *mf.Name
			metrics := mf.Metric
			for _, metric := range metrics {
				event := common.MapStr{}
				event["name"] = name

				labels := metric.Label

				if len(labels) != 0 {
					tagsMap := common.MapStr{}
					for _, label := range labels {
						if label.GetName() != "" && label.GetValue() != "" {
							tagsMap[label.GetName()] = label.GetValue()
						}
					}
					event["tags"] = tagsMap

				}

				counter := metric.GetCounter()
				if counter != nil {
					event["value"] = counter.GetValue()
				}

				guage := metric.GetGauge()
				if guage != nil {
					event["value"] = guage.GetValue()
				}

				summary := metric.GetSummary()
				if summary != nil {
					sum := strconv.FormatFloat(summary.GetSampleSum(), 'f', -1, 64)
					event["sum"] = sum

					count := strconv.FormatUint(summary.GetSampleCount(), 10)
					event["count"] = count

					quantiles := summary.GetQuantile()

					percentileMap := common.MapStr{}
					for _, quantile := range quantiles {
						key := strconv.FormatFloat(quantile.GetQuantile(), 'f', -1, 64)
						value := strconv.FormatFloat(quantile.GetValue(), 'f', -1, 64)
						percentileMap[key] = value
					}

					event["percentiles"] = percentileMap
				}

				histogram := metric.GetHistogram()
				if histogram != nil {
					sum := strconv.FormatUint(histogram.GetSampleCount(), 10)
					event["sum"] = sum

					count := strconv.FormatUint(histogram.GetSampleCount(), 10)
					event["count"] = count
					buckets := histogram.GetBucket()
					bucketMap := common.MapStr{}
					for _, bucket := range buckets {
						key := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
						value := strconv.FormatUint(bucket.GetCumulativeCount(), 10)
						bucketMap[key] = value
					}

					event["buckets"] = bucketMap
				}

				events = append(events, event)
			}
		}

	}

	return events, nil
}

func (m MetricSeter) Cleanup() error {
	return nil
}
