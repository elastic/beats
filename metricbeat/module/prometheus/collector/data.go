package collector

import (
	"math"
	"strconv"

	"github.com/elastic/beats/libbeat/common"

	dto "github.com/prometheus/client_model/go"
)

type PromEvent struct {
	key       string
	value     common.MapStr
	labels    common.MapStr
	labelHash string
}

func GetPromEventsFromMetricFamily(mf *dto.MetricFamily) []PromEvent {
	var events []PromEvent

	name := *mf.Name
	metrics := mf.Metric
	for _, metric := range metrics {
		event := PromEvent{
			key:       name,
			labelHash: "#",
		}
		value := common.MapStr{}
		labels := metric.Label

		if len(labels) != 0 {
			tagsMap := common.MapStr{}
			for _, label := range labels {
				if label.GetName() != "" && label.GetValue() != "" {
					tagsMap[label.GetName()] = label.GetValue()
				}
			}
			event.labels = tagsMap
			event.labelHash = tagsMap.String()

		}

		counter := metric.GetCounter()
		if counter != nil {
			value["value"] = int64(counter.GetValue())
		}

		gauge := metric.GetGauge()
		if gauge != nil {
			value["value"] = gauge.GetValue()
		}

		summary := metric.GetSummary()
		if summary != nil {
			value["sum"] = summary.GetSampleSum()
			value["count"] = summary.GetSampleCount()

			quantiles := summary.GetQuantile()

			percentileMap := common.MapStr{}
			for _, quantile := range quantiles {
				key := strconv.FormatFloat((100 * quantile.GetQuantile()), 'f', -1, 64)

				if math.IsNaN(quantile.GetValue()) == false {
					percentileMap[key] = quantile.GetValue()
				}

			}

			if len(percentileMap) != 0 {
				value["percentile"] = percentileMap
			}
		}

		histogram := metric.GetHistogram()
		if histogram != nil {
			value["sum"] = histogram.GetSampleSum()
			value["count"] = histogram.GetSampleCount()
			buckets := histogram.GetBucket()
			bucketMap := common.MapStr{}
			for _, bucket := range buckets {
				key := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
				bucketMap[key] = bucket.GetCumulativeCount()
			}

			value["bucket"] = bucketMap
		}

		event.value = value

		events = append(events, event)

	}
	return events
}
