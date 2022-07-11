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

package shipper

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	sc "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type shipper struct {
	log      *logp.Logger
	observer outputs.Observer
	conn     *grpc.ClientConn
	client   sc.ProducerClient
	timeout  time.Duration
	config   Config
}

func init() {
	outputs.RegisterType("shipper", makeShipper)
}

func makeShipper(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *conf.C,
) (outputs.Group, error) {

	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return outputs.Fail(err)
	}

	s := outputs.WithBackoff(&shipper{
		log:      logp.NewLogger("shipper"),
		observer: observer,
		config:   config,
		timeout:  config.Timeout,
	}, config.Backoff.Init, config.Backoff.Max)

	return outputs.Success(config.BulkMaxSize, config.MaxRetries, s)
}

// Connect establishes connection to the shipper server and implements `outputs.Connectable`.
func (c *shipper) Connect() error {
	tls, err := tlscommon.LoadTLSConfig(c.config.TLS)
	if err != nil {
		return fmt.Errorf("invalid shipper TLS configuration: %w", err)
	}

	var creds credentials.TransportCredentials
	if c.config.TLS != nil && c.config.TLS.Enabled != nil && *c.config.TLS.Enabled {
		creds = credentials.NewTLS(tls.ToConfig())
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: c.config.Timeout,
		}),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds),
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	c.log.Debugf("trying to connect to %s...", c.config.Server)

	conn, err := grpc.DialContext(ctx, c.config.Server, opts...)
	if err != nil {
		return fmt.Errorf("shipper connection failed with: %w", err)
	}
	c.log.Debugf("connect to %s established.", c.config.Server)

	c.conn = conn
	c.client = sc.NewProducerClient(conn)

	return nil
}

// Publish converts and sends a batch of events to the shipper server.
// Also, implements `outputs.Client`
func (c *shipper) Publish(ctx context.Context, batch publisher.Batch) error {
	if c.client == nil {
		return fmt.Errorf("connection is not established")
	}

	st := c.observer
	events := batch.Events()
	st.NewBatch(len(events))

	nonDroppedEvents := make([]publisher.Event, 0, len(events))
	convertedEvents := make([]*messages.Event, 0, len(events))

	c.log.Debugf("converting %d events to protobuf...", len(events))

	for i, e := range events {

		converted, err := toShipperEvent(e)
		if err != nil {
			// conversion errors are not recoverable, so we have to drop the event completely
			c.log.Errorf("%d/%d: %q, dropped", i+1, len(events), err)
			continue
		}

		convertedEvents = append(convertedEvents, converted)
		nonDroppedEvents = append(nonDroppedEvents, e)
	}

	droppedCount := len(events) - len(nonDroppedEvents)

	st.Dropped(droppedCount)
	c.log.Debugf("%d events converted to protobuf, %d dropped", len(nonDroppedEvents), droppedCount)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.client.PublishEvents(ctx, &messages.PublishRequest{
		Events: convertedEvents,
	})

	if status.Code(err) != codes.OK || resp == nil {
		batch.Cancelled()         // does not decrease the TTL
		st.Cancelled(len(events)) // we cancel the whole batch not just non-dropped events
		return fmt.Errorf("failed to publish the batch to the shipper, none of the %d events were accepted: %w", len(convertedEvents), err)
	}

	// with a correct server implementation should never happen, this error is not recoverable
	if int(resp.AcceptedCount) > len(nonDroppedEvents) {
		return fmt.Errorf(
			"server returned unexpected results, expected maximum accepted items %d, got %d",
			len(nonDroppedEvents),
			resp.AcceptedCount,
		)
	}

	// the server is supposed to retain the order of the initial events in the response
	// judging by the size of the result list we can determine what part of the initial
	// list was accepted and we can send the rest of the list for a retry
	retries := nonDroppedEvents[resp.AcceptedCount:]
	if len(retries) == 0 {
		batch.ACK()
		st.Acked(len(nonDroppedEvents))
		c.log.Debugf("%d events have been accepted, %d dropped", len(nonDroppedEvents), droppedCount)
	} else {
		batch.RetryEvents(retries) // decreases TTL unless guaranteed delivery
		st.Failed(len(retries))
		c.log.Debugf("%d events have been accepted, %d sent for retry, %d dropped", resp.AcceptedCount, len(retries), droppedCount)
	}

	return nil
}

// Close closes the connection to the shipper server.
// Also, implements `outputs.Client`
func (c *shipper) Close() error {
	if c.client == nil {
		return fmt.Errorf("connection is not established")
	}
	err := c.conn.Close()
	c.conn = nil
	c.client = nil

	return err
}

// String implements `outputs.Client`
func (c *shipper) String() string {
	return "shipper"
}

func convertMapStr(m mapstr.M) (*messages.Value, error) {
	if m == nil {
		return helpers.NewNullValue(), nil
	}

	fields := make(map[string]*messages.Value, len(m))

	for key, value := range m {
		var (
			protoValue *messages.Value
			err        error
		)
		switch v := value.(type) {
		case mapstr.M:
			protoValue, err = convertMapStr(v)
		default:
			protoValue, err = helpers.NewValue(v)
		}
		if err != nil {
			return nil, err
		}
		fields[key] = protoValue
	}

	s := &messages.Struct{
		Data: fields,
	}

	return helpers.NewStructValue(s), nil
}

func toShipperEvent(e publisher.Event) (*messages.Event, error) {
	meta, err := convertMapStr(e.Content.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to convert event metadata to protobuf: %w", err)
	}

	fields, err := convertMapStr(e.Content.Fields)
	if err != nil {
		return nil, fmt.Errorf("failed to convert event fields to protobuf: %w", err)
	}

	source := &messages.Source{}
	ds := &messages.DataStream{}

	inputIDVal, err := e.Content.Meta.GetValue("input_id")
	if err == nil {
		source.InputId, _ = inputIDVal.(string)
	}

	streamIDVal, err := e.Content.Meta.GetValue("stream_id")
	if err == nil {
		source.StreamId, _ = streamIDVal.(string)
	}

	dsType, err := e.Content.Fields.GetValue("data_stream.type")
	if err == nil {
		ds.Type, _ = dsType.(string)
	}
	dsNamespace, err := e.Content.Fields.GetValue("data_stream.namespace")
	if err == nil {
		ds.Namespace, _ = dsNamespace.(string)
	}
	dsDataset, err := e.Content.Fields.GetValue("data_stream.dataset")
	if err == nil {
		ds.Dataset, _ = dsDataset.(string)
	}

	return &messages.Event{
		Timestamp:  timestamppb.New(e.Content.Timestamp),
		Metadata:   meta.GetStructValue(),
		Fields:     fields.GetStructValue(),
		Source:     source,
		DataStream: ds,
	}, nil
}
