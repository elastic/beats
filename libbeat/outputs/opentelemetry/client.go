package opentelemetry

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
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

	okEvents, serialized := MapEvents(c.log,  data)
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
	resources := []resourcepb.Resource{}
	for _, event:= range data {
		resource := initResource(event.Content)
		resources = append(resources, resource)
	}
	return data, groupMetrics(resources, []*metricpb.InstrumentationLibraryMetrics{{InstrumentationLibrary: nil}})
}

func groupMetrics(resources []resourcepb.Resource, il []*metricpb.InstrumentationLibraryMetrics) []*metricpb.ResourceMetrics {
	metrics := make([]*metricpb.ResourceMetrics, len(resources))
	for i := range resources {
		metrics[i] = &metricpb.ResourceMetrics{
			Resource:                      &resources[i],
			InstrumentationLibraryMetrics: il,
		}
	}
	return metrics
}

func initResource(event beat.Event) resourcepb.Resource {
	resource := resourcepb.Resource{
		Attributes:             []*commonpb.KeyValue{},
		DroppedAttributesCount: 0,
	}
	timestamp, _ := event.Fields.GetValue("timestamp")
	keyVal:= commonpb.KeyValue {Key: "timestamp", Value:&commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", timestamp)}}}
	resource.Attributes = append(resource.Attributes, &keyVal )
	for name, val := range event.Meta {

		keyVal:= commonpb.KeyValue {Key: name, Value:&commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: fmt.Sprintf("%v", val)} }}
		resource.Attributes = append(resource.Attributes, &keyVal )

	}
return resource
}
