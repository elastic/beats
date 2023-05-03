package shipper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/go-units"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/shipper/tools"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/helpers"
	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
	"github.com/elastic/go-concert/unison"
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
// and one cursor input manager. It will create one or the other
// based on the config that is passed.
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
func (im *InputManager) Init(grp unison.Group, mode v2.Mode) error {
	im.log.Infof("initializing InputManager")
	return nil
}

// Create creates the input from a given config
func (im *InputManager) Create(cfg *config.C) (v2.Input, error) {
	config := Instance{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("error unpacking config: %w", err)
	}
	raw := mapstr.M{}
	err := cfg.Unpack(&raw)
	if err != nil {
		return nil, fmt.Errorf("error unpacking debug config: %w", err)
	}
	im.log.Infof("creating a new shipper input with config: %s", raw.String())
	im.log.Infof("parsed config as: %#v", config)
	//create a mapping of streams
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
	return &shipperInput{log: im.log, cfg: config, srvMut: &sync.Mutex{}, streams: streamDataMap}, nil
}

type shipperInput struct {
	log     *logp.Logger
	cfg     Instance
	streams map[string]streamData

	server     *grpc.Server
	shipper    *ShipperServer
	shipperSrv string
	srvMut     *sync.Mutex
}

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
// TODO: this needs to call Close() on all the clients
func (in *shipperInput) Stop() {
	in.log.Infof("shipper shutting down")
	if in.server != nil {
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

func (in *shipperInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	in.log.Infof("Running shipper input")
	// create clients ahead of time
	for streamID, streamProc := range in.streams {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			// TODO: need an EventListener?
			Processing: beat.ProcessingConfig{
				Processor: streamProc.processors,
			},
			CloseRef: inputContext.Cancelation,
		})
		if err != nil {
			return fmt.Errorf("error creating client for stream %s: %w", streamID, err)
		}
		in.log.Infof("Creating beat client for stream %s", streamID)

		newStreamData := streamData{client: client, index: in.streams[streamID].index, processors: in.streams[streamID].processors}
		in.streams[streamID] = newStreamData
	}

	//setup gRPC
	in.setupgRPC(pipeline)
	in.log.Infof("done setting up gRPC server")
	// wait for shutdown

	<-inputContext.Cancelation.Done()
	in.Stop()

	return nil
}

func (in *shipperInput) setupgRPC(pipeline beat.Pipeline) error {

	in.shipperSrv = strings.TrimPrefix(in.cfg.Conn.Server, "unix://")
	in.log.Infof("initializing grpc server at %s", in.shipperSrv)
	creds := insecure.NewCredentials()
	opts := []grpc.ServerOption{
		// TODO: figure out TLS
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

	// paranoid checking, make sure we have the base directory.
	dir := filepath.Dir(in.shipperSrv)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			in.srvMut.Unlock()
			return fmt.Errorf("could not create directory for unix socket %s: %w", dir, err)
		}
	}

	// on linux, net.Listen will fail if the file already exists
	if _, err = os.Stat(in.shipperSrv); err == nil {
		in.log.Debugf("listen address %s already exists, removing", in.shipperSrv)
		err = os.Remove(in.shipperSrv)
		if err != nil {
			in.srvMut.Unlock()
			return fmt.Errorf("error removing unix socket %s: %w", in.shipperSrv, err)
		}
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

	defer in.srvMut.Unlock()
	con, err := tools.DialTestAddr(in.shipperSrv)
	if err != nil {
		// this will stop the other go routine in the wait group
		in.server.Stop()
		return fmt.Errorf("failed to test connection with the gRPC server on %s: %w", in.shipperSrv, err)
	}
	_ = con.Close()

	return nil
}

func (in *shipperInput) sendEvent(event *messages.Event) (queue.EntryID, error) {
	//look for matching processor config
	stream, ok := in.streams[event.Source.StreamId]
	// should this be an error? can we continue on if there's no data stream associated with an event
	if !ok {
		return queue.EntryID(0), fmt.Errorf("could not find data stream associated with ID '%s'", event.Source.StreamId)
	}

	// inject index from stream config into metadata
	// not sure how stock beats do this, and also not sure if it's actually needed?
	metadata := helpers.AsMap(event.Metadata)
	metadata["index"] = stream.index

	// This will change if we move the events back to JSON.
	evt := beat.Event{
		Timestamp: event.Timestamp.AsTime(),
		Fields:    helpers.AsMap(event.Fields),
		Meta:      metadata,
	}
	in.log.Infof("metadata from incoming event: %s", evt.Meta.String())

	return stream.client.Publish(evt), nil
}
