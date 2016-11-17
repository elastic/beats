package publisher

import (
	"errors"
	"flag"
	"os"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/nranchev/go-libGeoIP"

	// load supported output plugins
	_ "github.com/elastic/beats/libbeat/outputs/console"
	_ "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/libbeat/outputs/fileout"
	_ "github.com/elastic/beats/libbeat/outputs/kafka"
	_ "github.com/elastic/beats/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/libbeat/outputs/redis"
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
	shipperName    string // Shipper name as set in the configuration file
	hostname       string // Host name as returned by the operation system
	name           string // The shipperName if configured, the hostname otherwise
	version        string
	IPAddrs        []string
	disabled       bool
	Index          string
	Output         []*outputWorker
	TopologyOutput outputs.TopologyOutputer
	geoLite        *libgeo.GeoIP
	Processors     *processors.Processors

	globalEventMetadata common.EventMetadata // Fields and tags to add to each event.

	RefreshTopologyTimer <-chan time.Time

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
	RefreshTopologyFreq  time.Duration      `config:"refresh_topology_freq"`
	TopologyExpire       int                `config:"topology_expire"`
	Geoip                common.Geoip       `config:"geoip"`

	// internal publisher queue sizes
	QueueSize     *int `config:"queue_size"`
	BulkQueueSize *int `config:"bulk_queue_size"`
	MaxProcs      *int `config:"max_procs"`
}

type Topology struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

const (
	DefaultQueueSize     = 1000
	DefaultBulkQueueSize = 0
)

func init() {
	publishDisabled = flag.Bool("N", false, "Disable actual publishing for testing")
}

func (publisher *BeatPublisher) IsPublisherIP(ip string) bool {
	for _, myip := range publisher.IPAddrs {
		if myip == ip {
			return true
		}
	}

	return false
}

func (publisher *BeatPublisher) GetServerName(ip string) string {
	// in case the IP is localhost, return current shipper name
	islocal, err := common.IsLoopback(ip)
	if err != nil {
		logp.Err("Parsing IP %s fails with: %s", ip, err)
		return ""
	}

	if islocal {
		return publisher.name
	}

	// find the shipper with the desired IP
	if publisher.TopologyOutput != nil {
		logp.Warn("Topology settings are deprecated.")
		return publisher.TopologyOutput.GetNameByIP(ip)
	}

	return ""
}

func (publisher *BeatPublisher) GeoLite() *libgeo.GeoIP {
	return publisher.geoLite
}

func (publisher *BeatPublisher) Connect() Client {
	atomic.AddUint32(&publisher.numClients, 1)
	return newClient(publisher)
}

func (publisher *BeatPublisher) UpdateTopologyPeriodically() {
	for range publisher.RefreshTopologyTimer {
		_ = publisher.PublishTopology() // ignore errors
	}
}

func (publisher *BeatPublisher) PublishTopology(params ...string) error {

	localAddrs := params
	if len(params) == 0 {
		addrs, err := common.LocalIPAddrsAsStrings(false)
		if err != nil {
			logp.Err("Getting local IP addresses fails with: %s", err)
			return err
		}
		localAddrs = addrs
	}

	if publisher.TopologyOutput != nil {
		debug("Add topology entry for %s: %s", publisher.name, localAddrs)

		err := publisher.TopologyOutput.PublishIPs(publisher.name, localAddrs)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create new PublisherType
func New(
	beatName string,
	beatVersion string,
	configs map[string]*common.Config,
	shipper ShipperConfig,
	processors *processors.Processors,
) (*BeatPublisher, error) {

	publisher := BeatPublisher{}
	err := publisher.init(beatName, beatVersion, configs, shipper, processors)
	if err != nil {
		return nil, err
	}
	return &publisher, nil
}

func (publisher *BeatPublisher) init(
	beatName string,
	beatVersion string,
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

	publisher.geoLite = common.LoadGeoIPData(shipper.Geoip)

	publisher.wsPublisher.Init()
	publisher.wsOutput.Init()

	if !publisher.disabled {
		plugins, err := outputs.InitOutputs(beatName, configs, shipper.TopologyExpire)
		if err != nil {
			return err
		}

		var outputers []*outputWorker
		var topoOutput outputs.TopologyOutputer
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

			if ok, _ := config.Bool("save_topology", 0); !ok {
				continue
			}

			topo, ok := output.(outputs.TopologyOutputer)
			if !ok {
				logp.Err("Output type %s does not support topology logging",
					plugin.Name)
				return errors.New("Topology output not supported")
			}

			if topoOutput != nil {
				logp.Err("Multiple outputs defined to store topology. " +
					"Please add save_topology = true option only for one output.")
				return errors.New("Multiple outputs defined to store topology")
			}

			topoOutput = topo
			logp.Info("Using %s to store the topology", plugin.Name)
		}

		publisher.Output = outputers
		publisher.TopologyOutput = topoOutput
	}

	if !publisher.disabled {
		if len(publisher.Output) == 0 {
			logp.Info("No outputs are defined. Please define one under the output section.")
			return errors.New("No outputs are defined. Please define one under the output section.")
		}

		if publisher.TopologyOutput == nil {
			logp.Debug("publish", "No output is defined to store the topology. The server fields might not be filled.")
		}
	}

	publisher.shipperName = shipper.Name
	publisher.hostname, err = os.Hostname()
	publisher.version = beatVersion
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

	if !publisher.disabled && publisher.TopologyOutput != nil {
		RefreshTopologyFreq := 10 * time.Second
		if shipper.RefreshTopologyFreq != 0 {
			RefreshTopologyFreq = shipper.RefreshTopologyFreq
		}
		publisher.RefreshTopologyTimer = time.Tick(RefreshTopologyFreq)
		logp.Info("Topology map refreshed every %s", RefreshTopologyFreq)

		// register shipper and its public IP addresses
		err = publisher.PublishTopology()
		if err != nil {
			logp.Err("Failed to publish topology: %s", err)
			return err
		}

		// update topology periodically
		go publisher.UpdateTopologyPeriodically()
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
