package publisher

import (
	"flag"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// command line flags
var publishDisabled *bool

var debug = logp.MakeDebug("publish")

type Context struct {
	publishOptions
	Signal op.Signaler
}

type publishOptions struct {
	Guaranteed bool
	Sync       bool
}

type TransactionalEventPublisher interface {
	PublishTransaction(transaction op.Signaler, events []common.MapStr)
}

type Publisher interface {
	Connect() Client
	ConnectX(beat.ClientConfig) (beat.Client, error)
	SetACKHandler(beat.PipelineACKHandler) error
}

type BeatPublisher struct {
	disabled bool
	name     string

	// keep count of clients connected to publisher. A publisher is allowed to
	// Stop only if all clients have been disconnected
	numClients atomic.Uint32

	pipeline beat.Pipeline
}

type ShipperConfig struct {
	common.EventMetadata `config:",inline"`     // Fields and tags to add to each event.
	Name                 string                 `config:"name"`
	Queue                common.ConfigNamespace `config:"queue"`

	// internal publisher queue sizes
	MaxProcs *int `config:"max_procs"`
}

func init() {
	publishDisabled = flag.Bool("N", false, "Disable actual publishing for testing")
}

// Create new PublisherType
func New(
	beat common.BeatInfo,
	output common.ConfigNamespace,
	shipper ShipperConfig,
	processors *processors.Processors,
) (*BeatPublisher, error) {
	publisher := BeatPublisher{}
	if err := publisher.init(beat, output, shipper, processors); err != nil {
		return nil, err
	}

	return &publisher, nil
}

func (publisher *BeatPublisher) init(
	beat common.BeatInfo,
	outConfig common.ConfigNamespace,
	shipper ShipperConfig,
	processors *processors.Processors,
) error {
	var err error
	publisher.disabled = *publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	publisher.name = shipper.Name
	if publisher.name == "" {
		publisher.name = beat.Hostname
	}

	publisher.pipeline, err = createPipeline(beat, shipper, processors, outConfig)
	if err != nil {
		return err
	}

	logp.Info("Publisher name: %s", publisher.name)
	return nil
}

func (publisher *BeatPublisher) Stop() {
	if publisher.numClients.Load() > 0 {
		panic("All clients must disconnect before shutting down publisher pipeline")
	}

	publisher.pipeline.Close()
}

func (publisher *BeatPublisher) Connect() Client {
	publisher.numClients.Inc()
	return newClient(publisher)
}

func (publisher *BeatPublisher) ConnectX(config beat.ClientConfig) (beat.Client, error) {
	return publisher.pipeline.ConnectWith(config)
}

func (publisher *BeatPublisher) SetACKHandler(h beat.PipelineACKHandler) error {
	return publisher.pipeline.SetACKHandler(h)
}

func (publisher *BeatPublisher) GetName() string {
	return publisher.name
}
