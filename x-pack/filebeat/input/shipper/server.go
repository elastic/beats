// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"

	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

// ShipperServer handles the actual gRPC server and associated connections
type ShipperServer struct {
	logger   *logp.Logger
	pipeline beat.Pipeline

	uuid string

	close *sync.Once
	ctx   context.Context
	stop  func()

	strictMode bool

	beatInput *shipperInput

	pb.UnimplementedProducerServer
}

// NewShipperServer creates a new server instance for handling gRPC endpoints.
// publisher can be set to nil, in which case the SetOutput() method must be called.
func NewShipperServer(pipeline beat.Pipeline, shipper *shipperInput) (*ShipperServer, error) {
	log := logp.NewLogger("shipper-server")

	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("error generating shipper UUID: %w", err)
	}

	srv := ShipperServer{
		uuid:       id.String(),
		logger:     log,
		pipeline:   pipeline,
		close:      &sync.Once{},
		beatInput:  shipper,
		strictMode: false,
	}

	srv.ctx, srv.stop = context.WithCancel(context.Background())

	return &srv, nil
}

// PublishEvents is the server implementation of the gRPC PublishEvents call.
func (serv *ShipperServer) PublishEvents(_ context.Context, req *messages.PublishRequest) (*messages.PublishReply, error) {
	resp := &messages.PublishReply{
		Uuid: serv.uuid,
	}
	// the value in the request is optional
	if req.Uuid != "" && req.Uuid != serv.uuid {
		serv.logger.Debugf("shipper UUID does not match, all events rejected. Expected = %s, actual = %s", serv.uuid, req.Uuid)
		return resp, status.Error(codes.FailedPrecondition, fmt.Sprintf("UUID does not match. Expected = %s, actual = %s", serv.uuid, req.Uuid))
	}

	if len(req.Events) == 0 {
		return nil, status.Error(codes.InvalidArgument, "publish request must contain at least one event")
	}

	if serv.strictMode {
		for _, e := range req.Events {
			err := serv.validateEvent(e)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
	}

	var accIdx uint64
	var err error
	for _, evt := range req.Events {
		accIdx, err = serv.beatInput.sendEvent(evt)
		if err != nil {
			serv.logger.Errorf("error sending event: %s", err)
		} else {
			resp.AcceptedCount++
		}

	}
	resp.AcceptedIndex = accIdx
	serv.logger.
		Debugf("finished publishing a batch. Events = %d, accepted = %d, accepted index = %d",
			len(req.Events),
			resp.AcceptedCount,
			resp.AcceptedIndex,
		)

	return resp, nil
}

// PersistedIndex implementation. Will track and send the oldest unacked event in the queue.
func (serv *ShipperServer) PersistedIndex(req *messages.PersistedIndexRequest, producer pb.Producer_PersistedIndexServer) error {
	serv.logger.Debug("new subscriber for persisted index change")
	defer serv.logger.Debug("unsubscribed from persisted index change")

	err := producer.Send(&messages.PersistedIndexReply{
		Uuid:           serv.uuid,
		PersistedIndex: serv.beatInput.acker.PersistedIndex(),
	})
	if err != nil {
		return fmt.Errorf("error sending index reply: %w", err)
	}

	pollingIntervalDur := req.PollingInterval.AsDuration()
	if pollingIntervalDur == 0 {
		return nil
	}

	ticker := time.NewTicker(pollingIntervalDur)
	defer ticker.Stop()

	for {
		select {
		case <-producer.Context().Done():
			return fmt.Errorf("producer context: %w", producer.Context().Err())

		case <-serv.ctx.Done():
			return fmt.Errorf("server is stopped: %w", serv.ctx.Err())

		case <-ticker.C:
			serv.logger.Infof("persistedIndex=%d", serv.beatInput.acker.PersistedIndex())
			err = producer.Send(&messages.PersistedIndexReply{
				Uuid:           serv.uuid,
				PersistedIndex: serv.beatInput.acker.PersistedIndex(),
			})
			if err != nil {
				return fmt.Errorf("failed to send the update: %w", err)
			}
		}
	}

}

// Close the server connection
func (serv *ShipperServer) Close() error {
	return nil
}

func (serv *ShipperServer) validateEvent(m *messages.Event) error {
	var msgs []string

	if err := m.Timestamp.CheckValid(); err != nil {
		msgs = append(msgs, fmt.Sprintf("timestamp: %s", err))
	}

	if err := serv.validateDataStream(m.DataStream); err != nil {
		msgs = append(msgs, fmt.Sprintf("datastream: %s", err))
	}

	if err := serv.validateSource(m.Source); err != nil {
		msgs = append(msgs, fmt.Sprintf("source: %s", err))
	}

	if len(msgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(msgs, "; "))
}

func (serv *ShipperServer) validateSource(s *messages.Source) error {
	if s == nil {
		return fmt.Errorf("cannot be nil")
	}

	var msgs []string
	if s.InputId == "" {
		msgs = append(msgs, "input_id is a required field")
	}

	if len(msgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(msgs, "; "))
}

func (serv *ShipperServer) validateDataStream(ds *messages.DataStream) error {
	if ds == nil {
		return fmt.Errorf("cannot be nil")
	}

	var msgs []string
	if ds.Dataset == "" {
		msgs = append(msgs, "dataset is a required field")
	}
	if ds.Namespace == "" {
		msgs = append(msgs, "namespace is a required field")
	}
	if ds.Type == "" {
		msgs = append(msgs, "type is a required field")
	}

	if len(msgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(msgs, "; "))
}
