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
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	v1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

const chunkSize = 1000

var instrumentationLibrary = &commonpb.InstrumentationLibrary{
	Name:    "beats",
	Version: "1.0.0",
}

func TestMapping(t *testing.T) {
	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"bar": 1}, Meta: common.MapStr{"hello": 1}}}

	for he, da := range event.Content.Fields {
		_ = he
		_ = da
	}
}

func TestValidate(t *testing.T) {
	var err error
	assert.Nil(t, err)
	ctx := context.Background()
	conn, outcont, err := createConnection(ctx, "0.0.0.0:4317")
	metricsClient := colmetricpb.NewMetricsServiceClient(conn)
	event := beat.Event{
		Timestamp:  time.Time{},
		Meta:       nil,
		Fields:     nil,
		Private:    nil,
		TimeSeries: false,
	}
	_ = event
	//metrics,_ := Adapt(event)
	metrics := Mock()
	lenMetrics := len(metrics)

	processed := 0
	var wg sync.WaitGroup
	for i := 0; i < lenMetrics; i += chunkSize {
		end := i + chunkSize
		if end > lenMetrics {
			end = lenMetrics
		}
		metricsBatch := metrics[i:end]
		wg.Add(1)
		go func(batch []*metricpb.ResourceMetrics) {
			out, err := metricsClient.Export(outcont, &colmetricpb.ExportMetricsServiceRequest{
				ResourceMetrics: batch,
			})
			_ = out
			if err != nil {
				_ = err
			} else {
				processed += len(batch)
			}
			wg.Done()
		}(metricsBatch)
	}
	wg.Wait()
}

func createConnection(ctx context.Context, endpoint string) (*grpc.ClientConn, context.Context, error) {
	outgoingCtx := metadata.NewOutgoingContext(ctx, nil)
	conn, err := grpc.DialContext(
		outgoingCtx,
		endpoint,
		grpc.WithInsecure(),
		//grpc.WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, context.Background(), fmt.Errorf("error creating grpc connection: %w", err)
	}

	return conn, outgoingCtx, nil
}

func Adapt(event beat.Event) ([]*metricpb.ResourceMetrics, error) {
	//timestamp, _ := event.Fields.GetValue("timestamp")

	resource := createResources(event)
	resources := []resourcepb.Resource{resource}
	return createArrayOfMetrics(resources, []*metricpb.InstrumentationLibraryMetrics{{InstrumentationLibrary: instrumentationLibrary}}), nil
}

func Mock() []*metricpb.ResourceMetrics {
	res := v1.Resource{
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
		DroppedAttributesCount: 0,
	}
	summary := metricpb.Summary{
		DataPoints: []*metricpb.SummaryDataPoint{
			{
				Attributes: []*commonpb.KeyValue{
					{
						Key:   "Namespace",
						Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "AWS/DynamoDB"}},
					},
					{
						Key:   "MetricName",
						Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "ConsumedReadCapacityUnits"}},
					}},
				StartTimeUnixNano: 1604948400000000000,
				TimeUnixNano:      1604948460000000000,
				Count:             1,
				Sum:               1.0,
				QuantileValues:    nil,
			},
		},
	}
	dasdfs := metricpb.Metric_Summary{
		Summary: &summary,
	}
	metrics := []*metricpb.Metric{
		{
			Name:        "amazonaws.com/AWS/DynamoDB/ConsumedReadCapacityUnits",
			Description: "",
			Unit:        "1",
			Data:        &dasdfs,
		},
	}
	lib := metricpb.InstrumentationLibraryMetrics{
		InstrumentationLibrary: instrumentationLibrary,
		Metrics:                metrics,
		SchemaUrl:              "",
	}
	sfs := metricpb.ResourceMetrics{
		Resource: &res,
		InstrumentationLibraryMetrics: []*metricpb.InstrumentationLibraryMetrics{
			&lib,
		},
		SchemaUrl: "",
	}

	return []*metricpb.ResourceMetrics{&sfs}
}
