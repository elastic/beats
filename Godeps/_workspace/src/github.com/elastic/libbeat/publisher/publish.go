package publisher

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/nranchev/go-libGeoIP"

	// load supported output plugins
	_ "github.com/elastic/libbeat/outputs/console"
	_ "github.com/elastic/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/libbeat/outputs/fileout"
	_ "github.com/elastic/libbeat/outputs/logstash"
	_ "github.com/elastic/libbeat/outputs/redis"
)

// command line flags
var publishDisabled *bool

var debug = logp.MakeDebug("publish")

// EventPublisher provides the interface for beats to publish events.
type eventPublisher interface {
	PublishEvent(ctx *context, event common.MapStr) bool
	PublishEvents(ctx *context, events []common.MapStr) bool
}

type context struct {
	publishOptions
	signal outputs.Signaler
}

type publishOptions struct {
	confirm bool
	sync    bool
}

type TransactionalEventPublisher interface {
	PublishTransaction(transaction outputs.Signaler, events []common.MapStr)
}

type PublisherType struct {
	shipperName    string // Shipper name as set in the configuration file
	hostname       string // Host name as returned by the operation system
	name           string // The shipperName if configured, the hostname otherwise
	ipaddrs        []string
	tags           []string
	disabled       bool
	Index          string
	Output         []*outputWorker
	TopologyOutput outputs.TopologyOutputer
	IgnoreOutgoing bool
	GeoLite        *libgeo.GeoIP

	RefreshTopologyTimer <-chan time.Time

	// wsOutput and wsPublisher should be used for proper shutdown of publisher
	// (not implemented yet). On shutdown the publisher should be finished first
	// and the outputers next, so no publisher will attempt to send messages on
	// closed channels.
	// Note: beat data producers must be shutdown before the publisher plugin
	wsOutput    workerSignal
	wsPublisher workerSignal

	syncPublisher  *syncPublisher
	asyncPublisher *asyncPublisher
}

type ShipperConfig struct {
	Name                  string
	Refresh_topology_freq int
	Ignore_outgoing       bool
	Topology_expire       int
	Tags                  []string
	Geoip                 common.Geoip
}

var Publisher PublisherType

type Topology struct {
	Name string `json:"name"`
	Ip   string `json:"ip"`
}

func init() {
	publishDisabled = flag.Bool("N", false, "Disable actual publishing for testing")
}

func PrintPublishEvent(event common.MapStr) {
	json, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		logp.Err("json.Marshal: %s", err)
	} else {
		debug("Publish: %s", string(json))
	}
}

func (publisher *PublisherType) IsPublisherIP(ip string) bool {
	for _, myip := range publisher.ipaddrs {
		if myip == ip {
			return true
		}
	}

	return false
}

func (publisher *PublisherType) GetServerName(ip string) string {
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
		return publisher.TopologyOutput.GetNameByIP(ip)
	}

	return ""
}

func (publisher *PublisherType) Client() Client {
	return &client{publisher}
}

func (publisher *PublisherType) UpdateTopologyPeriodically() {
	for _ = range publisher.RefreshTopologyTimer {
		_ = publisher.PublishTopology() // ignore errors
	}
}

func (publisher *PublisherType) PublishTopology(params ...string) error {

	localAddrs := params
	if len(params) == 0 {
		addrs, err := common.LocalIpAddrsAsStrings(false)
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

func (publisher *PublisherType) Init(
	beatName string,
	configs map[string]outputs.MothershipConfig,
	shipper ShipperConfig,
) error {
	var err error
	publisher.IgnoreOutgoing = shipper.Ignore_outgoing

	publisher.disabled = *publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	publisher.GeoLite = common.LoadGeoIPData(shipper.Geoip)

	publisher.wsOutput.Init()
	publisher.wsPublisher.Init()

	if !publisher.disabled {
		plugins, err := outputs.InitOutputs(beatName, configs, shipper.Topology_expire)
		if err != nil {
			return err
		}

		var outputers []*outputWorker
		var topoOutput outputs.TopologyOutputer
		for _, plugin := range plugins {
			output := plugin.Output
			config := plugin.Config

			debug("create output worker: %p, %p", config.Flush_interval, config.BulkMaxSize)

			outputers = append(outputers,
				newOutputWorker(config, output, &publisher.wsOutput, 1000))

			if !config.Save_topology {
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

		Publisher.Output = outputers
		Publisher.TopologyOutput = topoOutput
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
	if err != nil {
		return err
	}
	if len(publisher.shipperName) > 0 {
		publisher.name = publisher.shipperName
	} else {
		publisher.name = publisher.hostname
	}
	logp.Info("Publisher name: %s", publisher.name)

	publisher.tags = shipper.Tags

	//Store the publisher's IP addresses
	publisher.ipaddrs, err = common.LocalIpAddrsAsStrings(false)
	if err != nil {
		logp.Err("Failed to get local IP addresses: %s", err)
		return err
	}

	if !publisher.disabled && publisher.TopologyOutput != nil {
		RefreshTopologyFreq := 10 * time.Second
		if shipper.Refresh_topology_freq != 0 {
			RefreshTopologyFreq = time.Duration(shipper.Refresh_topology_freq) * time.Second
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

	publisher.asyncPublisher = newAsyncPublisher(publisher)
	publisher.syncPublisher = newSyncPublisher(publisher)

	return nil
}
