package publisher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/elasticsearch"
	"github.com/elastic/libbeat/outputs/fileout"
	"github.com/elastic/libbeat/outputs/redis"
	"github.com/nranchev/go-libGeoIP"
)

type PublisherType struct {
	name           string
	tags           string
	disabled       bool
	Index          string
	Output         []outputs.OutputInterface
	TopologyOutput outputs.OutputInterface
	IgnoreOutgoing bool
	GeoLite        *libgeo.GeoIP

	RefreshTopologyTimer <-chan time.Time
	Queue                chan common.MapStr
}

type AgentConfig struct {
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

var EnabledOutputPlugins map[outputs.OutputPlugin]outputs.OutputInterface = map[outputs.OutputPlugin]outputs.OutputInterface{
	outputs.RedisOutput:         new(redis.RedisOutput),
	outputs.ElasticsearchOutput: new(elasticsearch.ElasticsearchOutput),
	outputs.FileOutput:          new(fileout.FileOutput),
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
	// in case the IP is localhost, return current agent name
	islocal, err := common.IsLoopback(ip)
	if err != nil {
		logp.Err("Parsing IP %s fails with: %s", ip, err)
		return ""
	} else {
		if islocal {
			return publisher.name
		}
	}
	// find the agent with the desired IP
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

	// the @timestamp is mandatory
	ts, ok := event["@timestamp"].(common.Time)
	if !ok {
		return errors.New("Missing '@timestamp' field from event.")
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

	event["agent"] = publisher.name
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
				logp.Err("Fail to publish event type on output %s: %s", publisher.Output, err)
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

func (publisher *PublisherType) Init(publishDisabled bool,
	outputs map[string]outputs.MothershipConfig, agent AgentConfig) error {

	var err error
	publisher.IgnoreOutgoing = agent.Ignore_outgoing

	publisher.disabled = publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	publisher.GeoLite = common.LoadGeoIPData(agent.Geoip)

	for outputId, plugin := range EnabledOutputPlugins {
		outputName := outputId.String()
		output, exists := outputs[outputName]
		if exists && output.Enabled && !publisher.disabled {
			err := plugin.Init(output, agent.Topology_expire)
			if err != nil {
				logp.Err("Fail to initialize %s plugin as output: %s", outputName, err)
				return err
			}
			publisher.Output = append(publisher.Output, plugin)

			if output.Save_topology {
				if publisher.TopologyOutput != nil {
					logp.Err("Multiple outputs defined to store topology. Please add save_topology = true option only for one output.")
					return errors.New("Multiple outputs defined to store topology")
				}
				publisher.TopologyOutput = plugin
				logp.Info("Using %s to store the topology", outputName)
			}
		}
	}

	if !publisher.disabled {
		if len(publisher.Output) == 0 {
			logp.Info("No outputs are defined. Please define one under [output]")
			return errors.New("No outputs are define")
		}

		if publisher.TopologyOutput == nil {
			logp.Warn("No output is defined to store the topology. The server fields might not be filled.")
		}
	}

	publisher.name = agent.Name
	if len(publisher.name) == 0 {
		// use the hostname
		publisher.name, err = os.Hostname()
		if err != nil {
			return err
		}

		logp.Info("No agent name configured, using hostname '%s'", publisher.name)
	}

	if len(agent.Tags) > 0 {
		publisher.tags = strings.Join(agent.Tags, " ")
	}

	if !publisher.disabled && publisher.TopologyOutput != nil {
		RefreshTopologyFreq := 10 * time.Second
		if agent.Refresh_topology_freq != 0 {
			RefreshTopologyFreq = time.Duration(agent.Refresh_topology_freq) * time.Second
		}
		publisher.RefreshTopologyTimer = time.Tick(RefreshTopologyFreq)
		logp.Info("Topology map refreshed every %s", RefreshTopologyFreq)

		// register agent and its public IP addresses
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
