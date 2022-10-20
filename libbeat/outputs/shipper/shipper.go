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
	"sync"

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
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type pendingBatch struct {
	batch        publisher.Batch
	index        uint64
	serverID     string
	droppedCount int
}

type shipper struct {
	log      *logp.Logger
	observer outputs.Observer

	config   Config
	serverID string

	pending      []pendingBatch
	pendingMutex sync.Mutex

	conn        *grpc.ClientConn
	client      sc.ProducerClient
	clientMutex sync.Mutex

	backgroundCtx    context.Context
	backgroundCancel func()
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

	s := &shipper{
		log:      logp.NewLogger("shipper"),
		observer: observer,
		config:   config,
	}

	// for `Close` function to stop all the background work like acknowledgment loop
	s.backgroundCtx, s.backgroundCancel = context.WithCancel(context.Background())

	swb := outputs.WithBackoff(s, config.Backoff.Init, config.Backoff.Max)

	return outputs.Success(config.BulkMaxSize, config.MaxRetries, swb)
}

// Connect establishes connection to the shipper server and implements `outputs.Connectable`.
func (s *shipper) Connect() error {
	tls, err := tlscommon.LoadTLSConfig(s.config.TLS)
	if err != nil {
		return fmt.Errorf("invalid shipper TLS configuration: %w", err)
	}

	var creds credentials.TransportCredentials
	if s.config.TLS != nil && s.config.TLS.Enabled != nil && *s.config.TLS.Enabled {
		creds = credentials.NewTLS(tls.ToConfig())
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: s.config.Timeout,
		}),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds),
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	s.log.Debugf("trying to connect to %s...", s.config.Server)

	conn, err := grpc.DialContext(ctx, s.config.Server, opts...)
	if err != nil {
		return fmt.Errorf("shipper connection failed with: %w", err)
	}

	s.conn = conn
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	s.client = sc.NewProducerClient(conn)

	// we don't need a timeout context here anymore, we use the
	// `s.backgroundCtx` instead, it's going to be a long running client
	ackCtx, ackCancel := context.WithCancel(s.backgroundCtx)
	defer func() {
		// in case we return an error before we start the `ackLoop`
		// then we don't need this client anymore and must close the stream
		if err != nil {
			ackCancel()
		}
	}()

	indexClient, err := s.client.PersistedIndex(ackCtx, &messages.PersistedIndexRequest{
		PollingInterval: durationpb.New(s.config.AckPollingInterval),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to the server: %w", err)
	}
	indexReply, err := indexClient.Recv()
	if err != nil {
		return fmt.Errorf("failed to fetch server information: %w", err)
	}
	s.serverID = indexReply.GetUuid()

	s.log.Debugf("connection to %s (%s) established.", s.config.Server, s.serverID)

	go func() {
		defer ackCancel()
		s.log.Debugf("starting acknowledgment loop with server %s", s.serverID)
		// the loop returns only in case of error
		err := s.ackLoop(s.backgroundCtx, indexClient)
		s.log.Errorf("acknowledgment loop stopped: %s", err)
	}()

	return nil
}

// Publish converts and sends a batch of events to the shipper server.
// Also, implements `outputs.Client`
func (s *shipper) Publish(ctx context.Context, batch publisher.Batch) error {
	if s.client == nil {
		return fmt.Errorf("connection is not established")
	}

	st := s.observer
	events := batch.Events()
	st.NewBatch(len(events))

	toSend := make([]*messages.Event, 0, len(events))

	s.log.Debugf("converting %d events to protobuf...", len(events))

	droppedCount := 0

	for i, e := range events {
		converted, err := toShipperEvent(e)
		if err != nil {
			// conversion errors are not recoverable, so we have to drop the event completely
			s.log.Errorf("%d/%d: %q, dropped", i+1, len(events), err)
			droppedCount++
			continue
		}

		toSend = append(toSend, converted)
	}

	convertedCount := len(toSend)

	st.Dropped(droppedCount)
	s.log.Debugf("%d events converted to protobuf, %d dropped", convertedCount, droppedCount)

	var lastAcceptedIndex uint64

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	for len(toSend) > 0 {
		publishReply, err := s.client.PublishEvents(ctx, &messages.PublishRequest{
			Uuid:   s.serverID,
			Events: toSend,
		})

		if status.Code(err) != codes.OK {
			batch.Cancelled()         // does not decrease the TTL
			st.Cancelled(len(events)) // we cancel the whole batch not just non-dropped events
			return fmt.Errorf("failed to publish the batch to the shipper, none of the %d events were accepted: %w", len(toSend), err)
		}

		// with a correct server implementation should never happen, this error is not recoverable
		if int(publishReply.AcceptedCount) > len(toSend) {
			return fmt.Errorf(
				"server returned unexpected results, expected maximum accepted items %d, got %d",
				len(toSend),
				publishReply.AcceptedCount,
			)
		}
		toSend = toSend[publishReply.AcceptedCount:]
		lastAcceptedIndex = publishReply.AcceptedIndex
		s.log.Debugf("%d events have been accepted during a publish request", len(toSend))
	}

	s.log.Debugf("total of %d events have been accepted from batch, %d dropped", convertedCount, droppedCount)

	s.pendingMutex.Lock()
	s.pending = append(s.pending, pendingBatch{
		batch:        batch,
		index:        lastAcceptedIndex,
		serverID:     s.serverID,
		droppedCount: droppedCount,
	})
	s.pendingMutex.Unlock()

	return nil
}

// Close closes the connection to the shipper server.
// Also, implements `outputs.Client`
func (s *shipper) Close() error {
	if s.client == nil {
		return fmt.Errorf("connection is not established")
	}
	s.backgroundCancel()
	err := s.conn.Close()
	s.conn = nil
	s.client = nil
	s.pending = nil

	return err
}

// String implements `outputs.Client`
func (s *shipper) String() string {
	return "shipper"
}

func (s *shipper) ackLoop(ctx context.Context, ackClient sc.Producer_PersistedIndexClient) error {
	st := s.observer

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		default:
			// this sends an update and unblocks only if the `PersistedIndex` value has changed
			indexReply, err := ackClient.Recv()
			if err != nil {
				return fmt.Errorf("acknowledgment failed due to the connectivity error: %w", err)
			}

			s.pendingMutex.Lock()
			lastProcessed := 0
			for _, p := range s.pending {
				if p.serverID != indexReply.Uuid {
					s.log.Errorf("acknowledgment failed due to a connection to a different server %s, batch was accepted by %s", indexReply.Uuid, p.serverID)
					p.batch.Cancelled()
					st.Cancelled(len(p.batch.Events()))
					lastProcessed++
					continue
				}

				// if we met a batch that is ahead of the persisted index
				// we stop iterating and wait for another update from the server.
				// The latest pending batch has the max(AcceptedIndex).
				if p.index > indexReply.PersistedIndex {
					break
				}

				p.batch.ACK()
				ackedCount := len(p.batch.Events()) - p.droppedCount
				st.Acked(ackedCount)
				s.log.Debugf("%d events have been acknowledged, %d dropped", ackedCount, p.droppedCount)
				lastProcessed++
			}
			// so we don't perform any manipulation when the pending list is empty
			// or none of the batches were acknowledged by this persisted index update
			if lastProcessed != 0 {
				copy(s.pending[0:], s.pending[lastProcessed:])
				s.pending = s.pending[lastProcessed:]
			}
			s.pendingMutex.Unlock()
		}
	}
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
