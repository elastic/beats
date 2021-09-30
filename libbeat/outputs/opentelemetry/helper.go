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

package opentelemetry

import (
	"regexp"

	"github.com/elastic/beats/v7/libbeat/beat"

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
