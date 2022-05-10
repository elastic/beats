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
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/outputs"
	sc "github.com/elastic/beats/v7/libbeat/outputs/shipper/api"
	"github.com/elastic/beats/v7/libbeat/publisher"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const fieldShipperID = "shipper.eventId"

type shipper struct {
	log      *logp.Logger
	observer outputs.Observer
	conn     *grpc.ClientConn
	client   sc.ProducerClient
	timeout  time.Duration
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

	tls, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(fmt.Errorf("invalid shipper TLS configuration: %w", err))
	}

	var creds credentials.TransportCredentials
	if config.TLS != nil && config.TLS.Enabled != nil && *config.TLS.Enabled {
		creds = credentials.NewTLS(tls.ToConfig())
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: config.Timeout,
		}),
		grpc.WithTransportCredentials(creds),
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	log := logp.NewLogger("shipper")
	log.Debugf("trying to connect to %s...", config.Server)

	conn, err := grpc.DialContext(ctx, config.Server, opts...)
	if err != nil {
		return outputs.Fail(fmt.Errorf("shipper connection failed with: %w", err))
	}
	log.Debugf("connect to %s established.", config.Server)

	s := &shipper{
		log:      log,
		observer: observer,
		conn:     conn,
		client:   sc.NewProducerClient(conn),
		timeout:  config.Timeout,
	}

	return outputs.Success(config.BulkMaxSize, config.MaxRetries, s)
}

//nolint: godox // this implementation is not finished
func (c *shipper) Publish(ctx context.Context, batch publisher.Batch) error {
	st := c.observer
	events := batch.Events()
	st.NewBatch(len(events))

	dropped := 0

	grpcEvents := make([]*sc.Event, 0, len(events))

	c.log.Debugf("converting %d events to protobuf...", len(events))

	for i, e := range events {

		meta, err := convertMapStr(e.Content.Meta)
		if err != nil {
			// conversion errors are not recoverable, so we have to drop the event completely
			c.log.Errorf("%d/%d failed to convert event metadata to protobuf, dropped: %w", i+1, len(events), err)
			dropped++
			continue
		}

		fields, err := convertMapStr(e.Content.Fields)
		if err != nil {
			c.log.Errorf("%d/%d failed to convert event fields to protobuf, dropped: %w", i+1, len(events), err)
			dropped++
			continue
		}

		uuid, err := uuid.DefaultGenerator.NewV4()
		if err != nil {
			// this is unrecoverable and happens only when the random generator fails
			batch.Cancelled()
			return fmt.Errorf("failed to generate an event UUID: %w", err)
		}
		eventID := uuid.String()

		// storing for fast access when confirming the results
		_, err = e.Content.Meta.Put(fieldShipperID, eventID)
		if err != nil {
			c.log.Errorf("%d/%d failed to set shipper event ID, dropped: %w", i+1, len(events), err)
			dropped++
			continue
		}

		grpcEvents = append(grpcEvents, &sc.Event{
			EventId:   eventID,
			Timestamp: timestamppb.New(e.Content.Timestamp),
			Metadata:  meta.GetStructValue(),
			Fields:    fields.GetStructValue(),
			// TODO this contains temporary values, since they are required and not available from the event at the moment
			Input: &sc.Input{
				Id:   "beats",
				Name: "beats",
				Type: "beats",
			},
			// TODO this contains temporary values, since they are required and not propagated at the moment
			DataStream: &sc.DataStream{
				// Id:        "none", // not generated at the moment
				Type:      "shipper.output",
				Dataset:   "generic",
				Namespace: "default",
			},
		})
	}

	st.Dropped(dropped)
	c.log.Debugf("%d events converted to protobuf, %d dropped", len(grpcEvents), dropped)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.client.PublishEvents(ctx, &sc.PublishRequest{
		Events: grpcEvents,
	})

	if status.Code(err) != codes.OK {
		// if the response is nil, it means none of the events made it through due to a severe failure
		batch.Cancelled()
		st.Cancelled(len(events))
		c.log.Errorf("failed to publish the batch to the shipper, none of the %d events were accepted: %w", len(grpcEvents), err)
		return nil
	}

	retries := findRetries(resp, events)
	if len(retries) == 0 {
		batch.ACK()
		st.Acked(len(grpcEvents))
		c.log.Debugf("%d events have been acknowledged, %d dropped", len(grpcEvents), dropped)
	} else {
		batch.RetryEvents(retries)
		st.Failed(len(retries))
		c.log.Debugf("%d events have been acknowledged, %d sent for retry, %d dropped", len(grpcEvents), len(retries), dropped)
	}

	return nil
}

func (c *shipper) Close() error {
	return c.conn.Close()
}

func (c *shipper) String() string {
	return "shipper"
}

func findRetries(resp *sc.PublishReply, events []publisher.Event) []publisher.Event {
	if resp == nil || len(events) == 0 {
		return nil
	}

	// build a set of IDs for fast access below
	acceptedIDs := make(map[string]struct{}, len(resp.Results))
	for _, r := range resp.Results {
		acceptedIDs[r.EventId] = struct{}{}
	}

	retries := make([]publisher.Event, 0, len(events)-len(acceptedIDs))

	// we must range the original event list to retain the order
	for _, e := range events {
		// if the event has no shipper it was dropped, we just skip it
		eventID, err := e.Content.Meta.GetValue(fieldShipperID)
		if err != nil {
			continue
		}

		eventIDStr, ok := eventID.(string)
		if !ok {
			continue
		}

		_, ok = acceptedIDs[eventIDStr]
		if ok {
			continue
		}

		retries = append(retries, e)
	}

	return retries
}

func convertMapStr(m mapstr.M) (*structpb.Value, error) {
	if m == nil {
		return structpb.NewNullValue(), nil
	}

	fields := make(map[string]*structpb.Value, len(m))

	for key, value := range m {
		var (
			protoValue *structpb.Value
			err        error
		)
		switch v := value.(type) {
		case mapstr.M:
			protoValue, err = convertMapStr(v)
		default:
			protoValue, err = structpb.NewValue(v)
		}
		if err != nil {
			return nil, err
		}
		fields[key] = protoValue
	}

	s := &structpb.Struct{
		Fields: fields,
	}

	return structpb.NewStructValue(s), nil
}
