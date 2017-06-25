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
}

type BeatPublisher struct {
	shipperName string // Shipper name as set in the configuration file
	hostname    string // Host name as returned by the operation system
	name        string // The shipperName if configured, the hostname otherwise
	version     string

	disabled   bool
	processors *processors.Processors

	globalEventMetadata common.EventMetadata // Fields and tags to add to each event.

	// keep count of clients connected to publisher. A publisher is allowed to
	// Stop only if all clients have been disconnected
	numClients atomic.Uint32

	pipeline beat.Pipeline
}

type ShipperConfig struct {
	common.EventMetadata `config:",inline"` // Fields and tags to add to each event.
	Name                 string             `config:"name"`

	// internal publisher queue sizes
	QueueSize     *int `config:"queue_size"`
	BulkQueueSize *int `config:"bulk_queue_size"`
	MaxProcs      *int `config:"max_procs"`
}

const (
	DefaultQueueSize     = 1000
	DefaultBulkQueueSize = 0
)

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
	publisher.processors = processors

	publisher.disabled = *publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	shipper.InitShipperConfig()
	publisher.shipperName = shipper.Name
	publisher.hostname = beat.Hostname
	publisher.version = beat.Version
	if len(publisher.shipperName) > 0 {
		publisher.name = publisher.shipperName
	} else {
		publisher.name = publisher.hostname
	}

	publisher.pipeline, err = createPipeline(beat, shipper, outConfig)
	if err != nil {
		return err
	}

	logp.Info("Publisher name: %s", publisher.name)

	publisher.globalEventMetadata = shipper.EventMetadata
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

func (publisher *BeatPublisher) GetName() string {
	return publisher.name
}

func (config *ShipperConfig) InitShipperConfig() {

	// TODO: replace by ucfg
	if config.QueueSize == nil || *config.QueueSize <= 0 {
		queueSize := DefaultQueueSize
		config.QueueSize = &queueSize
	}

	if config.BulkQueueSize == nil || *config.BulkQueueSize < 0 {
		bulkQueueSize := DefaultBulkQueueSize
		config.BulkQueueSize = &bulkQueueSize
	}
}
