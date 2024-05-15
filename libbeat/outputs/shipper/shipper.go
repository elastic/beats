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
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"

	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	sc "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
	eventCount   int
	droppedCount int
}

type shipper struct {
	log      *logp.Logger
	observer outputs.Observer

	config Config

	conn      *grpc.ClientConn
	client    sc.ProducerClient
	ackClient sc.Producer_PersistedIndexClient

	serverID string

	// The publish function sends to ackLoopChan to notify the ack worker of
	// new pending batches
	ackBatchChan chan pendingBatch

	// The ack RPC listener sends to ackIndexChan to notify the ack worker
	// of the new persisted index
	ackIndexChan chan uint64

	// ackWaitGroup is used to synchronize the shutdown of the ack listener
	// and the ack worker when a connection is closed.
	ackWaitGroup sync.WaitGroup

	// ackCancel cancels the context for the ack listener and the ack worker,
	// notifying them to shut down.
	ackCancel context.CancelFunc
}

func init() {
	outputs.RegisterType("shipper", makeShipper)
}

// shipperProcessor serves as a wrapper for testing Publish() calls with alternate marshalling callbacks
var shipperProcessor = toShipperEvent

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

	swb := outputs.WithBackoff(s, config.Backoff.Init, config.Backoff.Max)

	return outputs.Group{
		Clients: []outputs.Client{swb},
		Retry:   config.MaxRetries,
		QueueFactory: memqueue.FactoryForSettings(
			memqueue.Settings{
				Events:        config.BulkMaxSize * 2,
				MaxGetRequest: config.BulkMaxSize,
				FlushTimeout:  0,
			}),
	}, nil
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

	s.log.Debugf("trying to connect to %s...", s.config.Server)

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, s.config.Server, opts...)
	if err != nil {
		return fmt.Errorf("shipper connection failed with: %w", err)
	}

	s.conn = conn
	s.client = sc.NewProducerClient(conn)

	return s.startACKLoop()
}

// Publish converts and sends a batch of events to the shipper server.
// Also, implements `outputs.Client`
func (s *shipper) Publish(ctx context.Context, batch publisher.Batch) error {
	err := s.publish(ctx, batch)
	if err != nil {
		// If there was an error then we are dropping our connection.
		s.Close()
	}
	return err
}

func (s *shipper) publish(ctx context.Context, batch publisher.Batch) error {
	if s.conn == nil {
		return fmt.Errorf("connection is not established")
	}

	events := batch.Events()
	s.observer.NewBatch(len(events))

	toSend := make([]*messages.Event, 0, len(events))

	s.log.Debugf("converting %d events to protobuf...", len(events))

	droppedCount := 0

	for i, e := range events {
		converted, err := shipperProcessor(e)
		if err != nil {
			// conversion errors are not recoverable, so we have to drop the event completely
			s.log.Errorf("%d/%d: %q, dropped", i+1, len(events), err)
			droppedCount++
			continue
		}

		toSend = append(toSend, converted)
	}

	convertedCount := len(toSend)

	s.observer.Dropped(droppedCount)
	s.log.Debugf("%d events converted to protobuf, %d dropped", convertedCount, droppedCount)

	var lastAcceptedIndex uint64

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	for len(toSend) > 0 {
		publishReply, err := s.client.PublishEvents(ctx, &messages.PublishRequest{
			Uuid:   s.serverID,
			Events: toSend,
		})

		if err != nil {
			if status.Code(err) == codes.ResourceExhausted {
				// This error can only come from the gRPC connection, and
				// most likely indicates this request exceeds the shipper's
				// RPC size limit. Split the batch if possible, otherwise we
				// need to drop it.
				if batch.SplitRetry() {
					// Report that we split a batch
					s.observer.Split()
				} else {
					batch.Drop()
					s.observer.Dropped(len(events))
					s.log.Errorf("dropping %d events because of RPC failure: %v", len(events), err)
				}
				return nil
			}
			// All other known errors are, in theory, retryable once the
			// RPC connection is successfully restarted, and don't depend on
			// the contents of the request. We should be cautious, though: if an
			// error is deterministic based on the contents of a publish
			// request, then cancelling here (instead of dropping or retrying)
			// will cause an infinite retry loop, wedging the pipeline.

			batch.Cancelled()                 // does not decrease the TTL
			s.observer.Cancelled(len(events)) // we cancel the whole batch not just non-dropped events
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
		s.log.Debugf("%d events have been accepted during a publish request", publishReply.AcceptedCount)
	}

	s.log.Debugf("total of %d events have been accepted from batch, %d dropped", convertedCount, droppedCount)

	// We've sent as much as we can to the shipper, release the batch's events and
	// save it in the queue of batches awaiting acknowledgment.
	batch.FreeEntries()
	s.ackBatchChan <- pendingBatch{
		batch:        batch,
		index:        lastAcceptedIndex,
		eventCount:   len(events),
		droppedCount: droppedCount,
	}

	return nil
}

// Close closes the connection to the shipper server.
// Also, implements `outputs.Client`
func (s *shipper) Close() error {
	if s.conn == nil {
		return fmt.Errorf("connection is not established")
	}
	s.ackCancel()
	s.ackWaitGroup.Wait()

	err := s.conn.Close()
	s.conn = nil
	s.client = nil

	return err
}

// String implements `outputs.Client`
func (s *shipper) String() string {
	return "shipper"
}

func (s *shipper) startACKLoop() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.ackCancel = cancel

	indexClient, err := s.client.PersistedIndex(ctx, &messages.PersistedIndexRequest{
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

	s.ackClient = indexClient
	s.ackBatchChan = make(chan pendingBatch)
	s.ackIndexChan = make(chan uint64)
	s.ackWaitGroup.Add(2)

	go func() {
		s.ackWorker(ctx)
		s.ackWaitGroup.Done()
	}()

	go func() {
		err := s.ackListener(ctx)
		s.ackWaitGroup.Done()
		if err != nil {
			s.log.Errorf("acknowledgment listener stopped: %s", err)

			// Shut down the connection and clear the output metadata.
			// This will not propagate back to the pipeline immediately,
			// but the next time Publish is called it will return an error
			// because there is no connection, which will signal the pipeline
			// to try to revive this output worker via Connect().
			s.Close()
		}
	}()

	return nil
}

// ackListener's only job is to listen to the persisted index RPC stream
// and forward its values to the ack worker.
func (s *shipper) ackListener(ctx context.Context) error {
	s.log.Debugf("starting acknowledgment listener with server %s", s.serverID)
	for {
		indexReply, err := s.ackClient.Recv()
		if err != nil {
			select {
			case <-ctx.Done():
				// If our context has been closed, this is an intentional closed
				// connection, so don't return the error.
				return nil
			default:
				// If the context itself is not closed then this means a real
				// connection error.
				return fmt.Errorf("ack listener closed connection: %w", err)
			}
		}
		s.ackIndexChan <- indexReply.PersistedIndex
	}
}

// ackWorker listens for newly published batches awaiting acknowledgment,
// and for new persisted indexes that should be forwarded to already-published
// batches.
func (s *shipper) ackWorker(ctx context.Context) {
	s.log.Debugf("starting acknowledgment loop with server %s", s.serverID)

	pending := []pendingBatch{}
	for {
		select {
		case <-ctx.Done():
			// If there are any pending batches left when the ack loop returns, then
			// they will never be acknowledged, so send the cancel signal.
			for _, p := range pending {
				p.batch.Cancelled()
			}
			return

		case newBatch := <-s.ackBatchChan:
			pending = append(pending, newBatch)

		case newIndex := <-s.ackIndexChan:
			lastProcessed := 0
			for _, p := range pending {
				// if we met a batch that is ahead of the persisted index
				// we stop iterating and wait for another update from the server.
				// The latest pending batch has the max(AcceptedIndex).
				if p.index > newIndex {
					break
				}

				p.batch.ACK()
				ackedCount := p.eventCount - p.droppedCount
				s.observer.Acked(ackedCount)
				s.log.Debugf("%d events have been acknowledged, %d dropped", ackedCount, p.droppedCount)
				lastProcessed++
			}
			// so we don't perform any manipulation when the pending list is empty
			// or none of the batches were acknowledged by this persisted index update
			if lastProcessed != 0 {
				remaining := len(pending) - lastProcessed
				copy(pending[0:], pending[lastProcessed:])
				pending = pending[:remaining]
			}
		}
	}
}

func toShipperEvent(e publisher.Event) (*messages.Event, error) {
	meta, err := helpers.NewValue(e.Content.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to convert event metadata to protobuf: %w", err)
	}

	fields, err := helpers.NewValue(e.Content.Fields)
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
