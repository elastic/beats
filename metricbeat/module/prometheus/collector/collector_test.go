// +build !integration

package collector

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestGetPromEventsFromMetricFamily(t *testing.T) {
	labels := common.MapStr{
		"handler": "query",
	}
	tests := []struct {
		Family *dto.MetricFamily
		Event  PromEvent
	}{
		{
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_COUNTER.Enum(),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  proto.String("handler"),
								Value: proto.String("query"),
							},
						},
						Counter: &dto.Counter{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: PromEvent{
				key: "http_request_duration_microseconds",
				value: common.MapStr{
					"value": int64(10),
				},
				labelHash: labels.String(),
				labels:    labels,
			},
		},
		{
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_GAUGE.Enum(),
				Metric: []*dto.Metric{
					{
						Gauge: &dto.Gauge{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: PromEvent{
				key: "http_request_duration_microseconds",
				value: common.MapStr{
					"value": float64(10),
				},
				labelHash: "#",
			},
		},
		{
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_SUMMARY.Enum(),
				Metric: []*dto.Metric{
					{
						Summary: &dto.Summary{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Quantile: []*dto.Quantile{
								{
									Quantile: proto.Float64(0.99),
									Value:    proto.Float64(10),
								},
							},
						},
					},
				},
			},
			Event: PromEvent{
				key: "http_request_duration_microseconds",
				value: common.MapStr{
					"count": uint64(10),
					"sum":   float64(10),
					"percentile": common.MapStr{
						"99": float64(10),
					},
				},
				labelHash: "#",
			},
		},
		{
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_HISTOGRAM.Enum(),
				Metric: []*dto.Metric{
					{
						Histogram: &dto.Histogram{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Bucket: []*dto.Bucket{
								{
									UpperBound:      proto.Float64(0.99),
									CumulativeCount: proto.Uint64(10),
								},
							},
						},
					},
				},
			},
			Event: PromEvent{
				key: "http_request_duration_microseconds",
				value: common.MapStr{
					"count": uint64(10),
					"sum":   float64(10),
					"bucket": common.MapStr{
						"0.99": uint64(10),
					},
				},
				labelHash: "#",
			},
		},
	}

	for _, test := range tests {
		event := GetPromEventsFromMetricFamily(test.Family)
		assert.Equal(t, len(event), 1)
		assert.Equal(t, event[0], test.Event)
	}
}
