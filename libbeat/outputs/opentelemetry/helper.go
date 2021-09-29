package opentelemetry

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"regexp"

	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

const (
	colNamespace = "namespace"
	colService   = "service"
	colPod       = "pod"
	colContainer = "container"
)

var regExpIsArray = regexp.MustCompilePOSIX(`\[((\"[a-zA-Z0-9\-\/._]+\")+,)*(\"[a-zA-Z0-9\-\/._]+\")\]`)

func createResourceFunc(event beat.Event) resourcepb.Resource {
	return resourcepb.Resource{
		Attributes: []*commonpb.KeyValue{
			{
				Key:   "pixie.cluster.id",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "hello"}},
			},
			{
				Key:   "instrumentation.provider",
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "sfsf"}},
			},
		},
	}

}

func createResources(event beat.Event) resourcepb.Resource {
	return createResourceFunc(event)
}

func createArrayOfMetrics(resources []resourcepb.Resource, il []*metricpb.InstrumentationLibraryMetrics) []*metricpb.ResourceMetrics {
	metrics := make([]*metricpb.ResourceMetrics, len(resources))
	for i := range resources {
		metrics[i] = &metricpb.ResourceMetrics{
			Resource:                      &resources[i],
			InstrumentationLibraryMetrics: il,
		}
	}
	return metrics
}
