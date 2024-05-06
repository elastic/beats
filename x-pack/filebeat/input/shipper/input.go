// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/shipper/tools"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
	"github.com/elastic/go-concert/unison"

	"github.com/docker/go-units"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	inputName = "shipper"
)

// Plugin registers the input
func Plugin(log *logp.Logger, _ inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:      inputName,
		Stability: feature.Experimental,
		Manager:   NewInputManager(log),
	}
}

// InputManager wraps one stateless input manager
type InputManager struct {
	log *logp.Logger
}

// NewInputManager creates a new shipper input manager
func NewInputManager(log *logp.Logger) *InputManager {
	log.Infof("creating new InputManager")
	return &InputManager{
		log: log.Named("shipper-beat"),
	}
}

// Init initializes the manager
// not sure if the shipper needs to do anything at this point?
func (im *InputManager) Init(_ unison.Group) error {
	return nil
}

// Create creates the input from a given config
// in an attempt to speed things up, this will create the processors from the config before we have access to the pipeline to create the clients
func (im *InputManager) Create(cfg *config.C) (v2.Input, error) {
	config := Instance{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("error unpacking config: %w", err)
	}
	// strip the config we get from agent
	config.Conn.Server = strings.TrimPrefix(config.Conn.Server, "unix://")
	// following lines are helpful for debugging config,
	// will be useful as we figure out how to integrate with agent

	// raw := mapstr.M{}
	// err := cfg.Unpack(&raw)
	// if err != nil {
	// 	return nil, fmt.Errorf("error unpacking debug config: %w", err)
	// }
	// im.log.Infof("creating a new shipper input with config: %s", raw.String())
	// im.log.Infof("parsed config as: %#v", config)

	// create a mapping of streams
	// when we get a new event, we use this to decide what processors to use
	streamDataMap := map[string]streamData{}
	for _, stream := range config.Input.Streams {
		// convert to an actual processor used by the client
		procList, err := processors.New(stream.Processors)
		if err != nil {
			return nil, fmt.Errorf("error creating processors for input: %w", err)
		}
		im.log.Infof("created processors for %s: %s", stream.ID, procList.String())
		streamDataMap[stream.ID] = streamData{index: stream.Index, processors: procList}
	}
	return &shipperInput{log: im.log, cfg: config, srvMut: &sync.Mutex{}, streams: streamDataMap, shipperSrv: config.Conn.Server, acker: newShipperAcker()}, nil
}

// shipperInput is the main runtime object for the shipper
type shipperInput struct {
	log     *logp.Logger
	cfg     Instance
	streams map[string]streamData

	server  *grpc.Server
	shipper *ShipperServer
	// TODO: we probably don't need this, and can just fetch the config
	shipperSrv string
	srvMut     *sync.Mutex

	acker *shipperAcker

	// incrementing counter that serves as an event ID
	eventIDInc uint64
}

// all the data associated with a given stream that the shipper needs access to.
type streamData struct {
	index      string
	client     beat.Client
	processors beat.ProcessorList
}

func (in *shipperInput) Name() string { return inputName }

func (in *shipperInput) Test(ctx v2.TestContext) error {
	return nil
}

// Stop the shipper
func (in *shipperInput) Stop() {
	in.log.Infof("shipper shutting down")
	// stop individual clients
	for streamID, stream := range in.streams {
		err := stream.client.Close()
		if err != nil {
			in.log.Infof("error closing client for stream: %s: %w", streamID, stream)
		}
	}
	in.srvMut.Lock()
	defer in.srvMut.Unlock()
	if in.shipper != nil {
		err := in.shipper.Close()
		if err != nil {
			in.log.Debugf("Error stopping shipper input: %s", err)
		}
		in.shipper = nil
	}
	if in.server != nil {
		in.server.GracefulStop()
		in.server = nil

	}
	err := os.Remove(in.shipperSrv)
	if err != nil {
		in.log.Debugf("error removing unix socket for grpc listener during shutdown: %s", err)
	}
}

// Run the shipper
func (in *shipperInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	in.log.Infof("Running shipper input")
	// create clients ahead of time
	for streamID, streamProc := range in.streams {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			PublishMode:   beat.GuaranteedSend,
			EventListener: acker.TrackingCounter(in.acker.Track),
			Processing: beat.ProcessingConfig{
				Processor:   streamProc.processors,
				DisableHost: true,
				DisableType: true,
			},
		})
		if err != nil {
			return fmt.Errorf("error creating client for stream %s: %w", streamID, err)
		}
		defer client.Close()
		in.log.Infof("Creating beat client for stream %s", streamID)

		newStreamData := streamData{client: client, index: in.streams[streamID].index, processors: in.streams[streamID].processors}
		in.streams[streamID] = newStreamData
	}

	// setup gRPC
	err := in.setupgRPC(pipeline)
	if err != nil {
		return fmt.Errorf("error starting shipper gRPC server: %w", err)
	}
	in.log.Infof("done setting up gRPC server")

	// wait for shutdown
	<-inputContext.Cancelation.Done()

	in.Stop()

	return nil
}

func (in *shipperInput) setupgRPC(pipeline beat.Pipeline) error {
	in.log.Infof("initializing grpc server at %s", in.shipperSrv)
	// Currently no TLS until we figure out mTLS issues in agent/shipper
	creds := insecure.NewCredentials()
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(64 * units.MiB),
	}

	var err error
	in.server = grpc.NewServer(opts...)
	in.shipper, err = NewShipperServer(pipeline, in)
	if err != nil {
		return fmt.Errorf("error creating shipper server: %w", err)
	}

	pb.RegisterProducerServer(in.server, in.shipper)

	in.srvMut.Lock()

	// treat most of these checking errors as "soft" errors
	// Try to make the environment clean, but trust newListener() to fail if it can't just start.

	// paranoid checking, make sure we have the base directory.
	dir := filepath.Dir(in.shipperSrv)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		in.log.Warnf("could not create directory for unix socket %s: %w", dir, err)
	}

	// on linux, net.Listen will fail if the file already exists
	err = os.Remove(in.shipperSrv)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		in.log.Warnf("could not remove pre-existing socket at %s: %w", in.shipperSrv, err)
	}

	lis, err := newListener(in.log, in.shipperSrv)
	if err != nil {
		in.srvMut.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", in.shipperSrv, err)
	}

	go func() {
		in.log.Infof("gRPC listening on %s", in.shipperSrv)
		err = in.server.Serve(lis)
		if err != nil {
			in.log.Errorf("gRPC server shut down with error: %s", err)
		}
	}()

	// make sure connection is up before mutex is released;
	// if close() on the socket is called before it's started, it will trigger a race.
	defer in.srvMut.Unlock()
	con, err := tools.DialTestAddr(in.shipperSrv, in.cfg.Conn.InitialTimeout)
	if err != nil {
		// this will stop the other go routine in the wait group
		in.server.Stop()
		return fmt.Errorf("failed to test connection with the gRPC server on %s: %w", in.shipperSrv, err)
	}
	_ = con.Close()

	return nil
}

func (in *shipperInput) sendEvent(event *messages.Event) (uint64, error) {
	// look for matching processor config
	stream, ok := in.streams[event.Source.StreamId]
	if !ok {
		return 0, fmt.Errorf("could not find data stream associated with ID '%s'", event.Source.StreamId)
	}

	evt := beat.Event{
		Timestamp: event.Timestamp.AsTime(),
		Fields:    helpers.AsMap(event.Fields),
		Meta:      helpers.AsMap(event.Metadata),
	}
	atomic.AddUint64(&in.eventIDInc, 1)

	stream.client.Publish(evt)

	return in.eventIDInc, nil
}
