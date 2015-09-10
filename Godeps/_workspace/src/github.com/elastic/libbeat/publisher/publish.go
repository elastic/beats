package publisher

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/nranchev/go-libGeoIP"

	// load supported output plugins
	_ "github.com/elastic/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/libbeat/outputs/fileout"
	_ "github.com/elastic/libbeat/outputs/lumberjack"
	_ "github.com/elastic/libbeat/outputs/redis"
)

// command line flags
var publishDisabled *bool

type PublisherType struct {
	name           string
	tags           []string
	disabled       bool
	Index          string
	Output         []outputs.Outputer
	TopologyOutput outputs.TopologyOutputer
	IgnoreOutgoing bool
	GeoLite        *libgeo.GeoIP

	RefreshTopologyTimer <-chan time.Time
	Queue                chan common.MapStr
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
		logp.Debug("publish", "Publish: %s", string(json))
	}
}

func (publisher *PublisherType) GetServerName(ip string) string {
	// in case the IP is localhost, return current shipper name
	islocal, err := common.IsLoopback(ip)
	if err != nil {
		logp.Err("Parsing IP %s fails with: %s", ip, err)
		return ""
	} else {
		if islocal {
			return publisher.name
		}
	}
	// find the shipper with the desired IP
	if publisher.TopologyOutput != nil {
		return publisher.TopologyOutput.GetNameByIP(ip)
	} else {
		return ""
	}
}

func (publisher *PublisherType) publishFromQueue() {
	for mapstr := range publisher.Queue {
		err := publisher.publishEvent(mapstr)
		if err != nil {
			logp.Err("Publishing failed: %v", err)
		}
	}
}

func (publisher *PublisherType) publishEvent(event common.MapStr) error {

	// the timestamp is mandatory
	ts, ok := event["timestamp"].(common.Time)
	if !ok {
		return errors.New("Missing 'timestamp' field from event.")
	}

	// the count is mandatory
	err := event.EnsureCountField()
	if err != nil {
		return err
	}

	// the type is mandatory
	_, ok = event["type"].(string)
	if !ok {
		return errors.New("Missing 'type' field from event.")
	}

	var src_server, dst_server string
	src, ok := event["src"].(*common.Endpoint)
	if ok {
		src_server = publisher.GetServerName(src.Ip)
		event["client_ip"] = src.Ip
		event["client_port"] = src.Port
		event["client_proc"] = src.Proc
		event["client_server"] = src_server
		delete(event, "src")
	}
	dst, ok := event["dst"].(*common.Endpoint)
	if ok {
		dst_server = publisher.GetServerName(dst.Ip)
		event["ip"] = dst.Ip
		event["port"] = dst.Port
		event["proc"] = dst.Proc
		event["server"] = dst_server
		delete(event, "dst")
	}

	if publisher.IgnoreOutgoing && dst_server != "" &&
		dst_server != publisher.name {
		// duplicated transaction -> ignore it
		logp.Debug("publish", "Ignore duplicated transaction on %s: %s -> %s", publisher.name, src_server, dst_server)
		return nil
	}

	event["shipper"] = publisher.name
	if len(publisher.tags) > 0 {
		event["tags"] = publisher.tags
	}

	if publisher.GeoLite != nil {
		real_ip, exists := event["real_ip"]
		if exists && len(real_ip.(string)) > 0 {
			loc := publisher.GeoLite.GetLocationByIP(real_ip.(string))
			if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
				event["client_location"] = fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
			}
		} else {
			if len(src_server) == 0 && src != nil { // only for external IP addresses
				loc := publisher.GeoLite.GetLocationByIP(src.Ip)
				if loc != nil && loc.Latitude != 0 && loc.Longitude != 0 {
					event["client_location"] = fmt.Sprintf("%f, %f", loc.Latitude, loc.Longitude)
				}
			}
		}
	}

	if logp.IsDebug("publish") {
		PrintPublishEvent(event)
	}

	// add transaction
	has_error := false
	if !publisher.disabled {
		for i := 0; i < len(publisher.Output); i++ {
			err := publisher.Output[i].PublishEvent(time.Time(ts), event)
			if err != nil {
				logp.Err("Fail to publish event type on output %s: %v", publisher.Output[i], err)
				has_error = true
			}
		}
	}

	if has_error {
		return errors.New("Fail to publish event")
	}
	return nil
}

func (publisher *PublisherType) UpdateTopologyPeriodically() {
	for _ = range publisher.RefreshTopologyTimer {
		publisher.PublishTopology()
	}
}

func (publisher *PublisherType) PublishTopology(params ...string) error {

	var localAddrs []string = params

	if len(params) == 0 {
		addrs, err := common.LocalIpAddrsAsStrings(false)
		if err != nil {
			logp.Err("Getting local IP addresses fails with: %s", err)
			return err
		}
		localAddrs = addrs
	}

	if publisher.TopologyOutput != nil {
		logp.Debug("publish", "Add topology entry for %s: %s", publisher.name, localAddrs)

		err := publisher.TopologyOutput.PublishIPs(publisher.name, localAddrs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (publisher *PublisherType) Init(
	beat string,
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

	if !publisher.disabled {
		plugins, err := outputs.InitOutputs(beat, configs, shipper.Topology_expire)
		if err != nil {
			return err
		}

		var outputers []outputs.Outputer = nil
		var topoOutput outputs.TopologyOutputer = nil
		for _, plugin := range plugins {
			output := plugin.Output
			config := plugin.Config
			outputers = append(outputers, output)

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
			logp.Warn("No output is defined to store the topology. The server fields might not be filled.")
		}
	}

	publisher.name = shipper.Name
	if len(publisher.name) == 0 {
		// use the hostname
		publisher.name, err = os.Hostname()
		if err != nil {
			return err
		}

		logp.Info("No shipper name configured, using hostname '%s'", publisher.name)
	}

	publisher.tags = shipper.Tags

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

	publisher.Queue = make(chan common.MapStr, 1000)
	go publisher.publishFromQueue()

	return nil
}
