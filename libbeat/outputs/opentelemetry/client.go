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
	"time"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type publishFn func(
	keys outil.Selector,
	data []publisher.Event,
) ([]publisher.Event, error)

type client struct {
	log           *logp.Logger
	observer      outputs.Observer
	index         string
	key           outil.Selector
	timeout       time.Duration
	metricsClient colmetricpb.MetricsServiceClient
	logsClient    *collogspb.LogsServiceClient
	outgoingCtx   context.Context
}

func newClient(
	observer outputs.Observer,
	timeout time.Duration,
) *client {
	return &client{
		log:      logp.NewLogger("opentelemetry"),
		observer: observer,
		timeout:  timeout,
	}
}

func (c *client) Connect() error {
	c.log.Debug("connect")
	ctx := context.Background()
	conn, outgoingCtx, err := initConnection(ctx, "0.0.0.0:4317")
	metricsClient := colmetricpb.NewMetricsServiceClient(conn)
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	c.metricsClient = metricsClient
	c.outgoingCtx = outgoingCtx
	return err
}

func initConnection(ctx context.Context, endpoint string) (*grpc.ClientConn, context.Context, error) {
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

func (c *client) Close() error {
	c.log.Debug("close connection")
	return nil
}

func (c *client) Publish(_ context.Context, batch publisher.Batch) error {
	if c == nil {
		panic("no client")
	}
	if batch == nil {
		panic("no batch")
	}

	events := batch.Events()
	c.observer.NewBatch(len(events))
	rest, err := c.publish(c.key, events)
	if rest != nil {
		c.observer.Failed(len(rest))
		batch.RetryEvents(rest)
		return err
	}

	batch.ACK()
	return err
}

func (c *client) String() string {
	return "opentelemetry"
}

func (c *client) publish(key outil.Selector, data []publisher.Event) ([]publisher.Event, error) {
	var okEvents []publisher.Event

	okEvents, serialized := MapEvents(c.log, data)
	c.observer.Dropped(len(data) - len(okEvents))
	if len(serialized) == 0 {
		return nil, nil
	}

	data = okEvents[:0]
	dropped := 0
	failed := data[:0]
	result, err := c.metricsClient.Export(c.outgoingCtx, &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: serialized,
	})
	_ = result
	if err != nil {
		failed = append(failed, data...)
	}
	c.observer.Dropped(dropped)
	c.observer.Acked(len(okEvents) - len(failed))
	return failed, err
}

func MapEvents(log *logp.Logger, data []publisher.Event) ([]publisher.Event, []*metricpb.ResourceMetrics) {
	metrics := []*metricpb.ResourceMetrics{}
	for _, event := range data {
		resource := initResource(event.Content)
		lib := initMetrics(event.Content)

		metric := metricpb.ResourceMetrics{
			Resource: &resource,
			InstrumentationLibraryMetrics: []*metricpb.InstrumentationLibraryMetrics{
				&lib,
			},
			SchemaUrl: "",
		}
		metrics = append(metrics, &metric)
	}

	return data, metrics
}

func initMetrics(event beat.Event) metricpb.InstrumentationLibraryMetrics {
	metrics := []*metricpb.Metric{}
	time := event.Timestamp.UnixNano()
	gauge := metricpb.Gauge{}
	for key, val := range event.Fields {
		switch val.(type) {
		case int, int64:
			var i = val.(int)
			gauge = metricpb.Gauge{
				DataPoints: []*metricpb.NumberDataPoint{
					{

						StartTimeUnixNano: uint64(time),
						TimeUnixNano:      uint64(time),
						Value:             &metricpb.NumberDataPoint_AsInt{AsInt: int64(i)},
					},
				},
			}
		case float64, common.Float:
			gauge = metricpb.Gauge{
				DataPoints: []*metricpb.NumberDataPoint{
					{

						StartTimeUnixNano: uint64(time),
						TimeUnixNano:      uint64(time),
						Value:             &metricpb.NumberDataPoint_AsDouble{AsDouble: val.(float64)},
					},
				},
			}
		default:
		}
		if len(gauge.DataPoints) > 0 {
			gaugeMetric := metricpb.Metric_Gauge{
				Gauge: &gauge,
			}
			metric := metricpb.Metric{

				Name:        key,
				Description: "",
				Unit:        "perc",
				Data:        &gaugeMetric,
			}
			metrics = append(metrics, &metric)
		}

	}

	lib := metricpb.InstrumentationLibraryMetrics{
		InstrumentationLibrary: nil,
		Metrics:                metrics,
		SchemaUrl:              "",
	}
	return lib
}

func initResource(event beat.Event) resourcepb.Resource {
	resource := resourcepb.Resource{
		Attributes:             []*commonpb.KeyValue{},
		DroppedAttributesCount: 0,
	}
	timestamp, _ := event.Fields.GetValue("timestamp")
	keyVal := commonpb.KeyValue{Key: "timestamp", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", timestamp)}}}
	resource.Attributes = append(resource.Attributes, &keyVal)
	for name, val := range event.Meta {

		keyVal := commonpb.KeyValue{Key: name, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", val)}}}
		resource.Attributes = append(resource.Attributes, &keyVal)

	}
	for name, val := range event.Fields {
		switch val.(type) {
		case string, bool:
			keyVal := commonpb.KeyValue{Key: name, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", val)}}}
			resource.Attributes = append(resource.Attributes, &keyVal)
		}

	}
	return resource
}
