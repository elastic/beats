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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/gofrs/uuid"

	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

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
func (serv *ShipperServer) PublishEvents(ctx context.Context, req *messages.PublishRequest) (*messages.PublishReply, error) {
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

	var accIdx queue.EntryID
	var err error
	for _, evt := range req.Events {
		accIdx, err = serv.beatInput.sendEvent(evt)
		if err != nil {
			serv.logger.Errorf("error sending event: %s", err)
		} else {
			resp.AcceptedCount++
		}
		//TODO: There's probably a better way to track accepted count, presumably using the callbacks in the beat client

	}
	resp.AcceptedIndex = uint64(accIdx)
	serv.logger.
		Debugf("finished publishing a batch. Events = %d, accepted = %d, accepted index = %d",
			len(req.Events),
			resp.AcceptedCount,
			//TODO: we don't get an index like we do from TryPublish in the queue. Should we use anything else?
			resp.AcceptedIndex,
		)

	return resp, nil
}

// PersistedIndex implementation. Will track and send the oldest unacked event in the queue.
func (serv *ShipperServer) PersistedIndex(req *messages.PersistedIndexRequest, producer pb.Producer_PersistedIndexServer) error {
	//TODO: not yet sure how to implement this in the beat
	// on the old shipper this is done via the PersistedIndex() call from the raw queue interface, which I don't think we have access to here

	serv.logger.Debug("new subscriber for persisted index change")
	defer serv.logger.Debug("unsubscribed from persisted index change")
	idx, err := serv.pipeline.PersistedIndex()
	if err != nil {
		return fmt.Errorf("error fetching persisted index from pipeline: %w", err)
	}
	err = producer.Send(&messages.PersistedIndexReply{
		Uuid:           serv.uuid,
		PersistedIndex: uint64(idx),
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
			persistedIndex, err := serv.pipeline.PersistedIndex()
			if err != nil {
				return fmt.Errorf("error fetching persisted index from pipeline: %w", err)
			}
			serv.logger.Infof("sending PersistedIndex reply. ID=%d", persistedIndex)
			err = producer.Send(&messages.PersistedIndexReply{
				Uuid:           serv.uuid,
				PersistedIndex: uint64(persistedIndex),
			})
			if err != nil {
				return fmt.Errorf("failed to send the update: %w", err)
			}
		}
	}

}

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
