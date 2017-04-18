package publisher

import (
	"errors"
	"flag"
	"sync/atomic"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"

	// load supported output plugins
	_ "github.com/elastic/beats/libbeat/outputs/console"
	_ "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/libbeat/outputs/fileout"
	_ "github.com/elastic/beats/libbeat/outputs/kafka"
	_ "github.com/elastic/beats/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/libbeat/outputs/redis"

	// load support output codec
	_ "github.com/elastic/beats/libbeat/outputs/codecs/format"
	_ "github.com/elastic/beats/libbeat/outputs/codecs/json"
)

// command line flags
var publishDisabled *bool

var debug = logp.MakeDebug("publish")

type Context struct {
	publishOptions
	Signal op.Signaler
}

type pipeline interface {
	publish(m message) bool
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
	IPAddrs     []string
	disabled    bool
	Index       string
	Output      []*outputWorker
	Processors  *processors.Processors

	globalEventMetadata common.EventMetadata // Fields and tags to add to each event.

	// On shutdown the publisher is finished first and the outputers next,
	// so no publisher will attempt to send messages on closed channels.
	// Note: beat data producers must be shutdown before the publisher plugin
	wsPublisher workerSignal
	wsOutput    workerSignal

	pipelines struct {
		sync  pipeline
		async pipeline
	}

	// keep count of clients connected to publisher. A publisher is allowed to
	// Stop only if all clients have been disconnected
	numClients uint32
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

func (publisher *BeatPublisher) Connect() Client {
	atomic.AddUint32(&publisher.numClients, 1)
	return newClient(publisher)
}

func (publisher *BeatPublisher) GetName() string {
	return publisher.name
}

// Create new PublisherType
func New(
	beat common.BeatInfo,
	configs map[string]*common.Config,
	shipper ShipperConfig,
	processors *processors.Processors,
) (*BeatPublisher, error) {

	publisher := BeatPublisher{}
	err := publisher.init(beat, configs, shipper, processors)
	if err != nil {
		return nil, err
	}
	return &publisher, nil
}

func (publisher *BeatPublisher) init(
	beat common.BeatInfo,
	configs map[string]*common.Config,
	shipper ShipperConfig,
	processors *processors.Processors,
) error {
	var err error
	publisher.Processors = processors

	publisher.disabled = *publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	shipper.InitShipperConfig()

	publisher.wsPublisher.Init()
	publisher.wsOutput.Init()

	if !publisher.disabled {
		plugins, err := outputs.InitOutputs(beat, configs)

		if err != nil {
			return err
		}

		var outputers []*outputWorker
		for _, plugin := range plugins {
			output := plugin.Output
			config := plugin.Config

			debug("Create output worker")

			outputers = append(outputers,
				newOutputWorker(
					config,
					output,
					&publisher.wsOutput,
					*shipper.QueueSize,
					*shipper.BulkQueueSize))

		}

		publisher.Output = outputers
	}

	if !publisher.disabled {
		if len(publisher.Output) == 0 {
			logp.Info("No outputs are defined. Please define one under the output section.")
			return errors.New("No outputs are defined. Please define one under the output section.")
		}
	}

	publisher.shipperName = shipper.Name
	publisher.hostname = beat.Hostname
	publisher.version = beat.Version
	if err != nil {
		return err
	}
	if len(publisher.shipperName) > 0 {
		publisher.name = publisher.shipperName
	} else {
		publisher.name = publisher.hostname
	}
	logp.Info("Publisher name: %s", publisher.name)

	publisher.globalEventMetadata = shipper.EventMetadata

	//Store the publisher's IP addresses
	publisher.IPAddrs, err = common.LocalIPAddrsAsStrings(false)
	if err != nil {
		logp.Err("Failed to get local IP addresses: %s", err)
		return err
	}

	publisher.pipelines.async = newAsyncPipeline(publisher, *shipper.QueueSize, *shipper.BulkQueueSize, &publisher.wsPublisher)
	publisher.pipelines.sync = newSyncPipeline(publisher, *shipper.QueueSize, *shipper.BulkQueueSize)
	return nil
}

func (publisher *BeatPublisher) Stop() {
	if atomic.LoadUint32(&publisher.numClients) > 0 {
		panic("All clients must disconnect before shutting down publisher pipeline")
	}

	publisher.wsPublisher.stop()
	publisher.wsOutput.stop()
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
