package main

import (
	"encoding/json"
	"errors"
	"os"
	"packetbeat/logp"
	"packetbeat/outputs"
	"strings"
	"time"
)

type PublisherType struct {
	name                string
	tags                string
	disabled            bool
	Index               string
	Output              []outputs.OutputInterface
	TopologyOutput      outputs.OutputInterface
	ElasticsearchOutput outputs.ElasticsearchOutputType
	RedisOutput         outputs.RedisOutputType
	FileOutput          outputs.FileOutputType

	RefreshTopologyTimer <-chan time.Time
}

var Publisher PublisherType

// Config
type tomlAgent struct {
	Name                  string
	Refresh_topology_freq int
	Ignore_outgoing       bool
	Topology_expire       int
	Tags                  []string
}

type Topology struct {
	Name string `json:"name"`
	Ip   string `json:"ip"`
}

func PrintPublishEvent(event *outputs.Event) {
	json, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		logp.Err("json.Marshal: %s", err)
	} else {
		logp.Debug("publish", "Publish: %s", string(json))
	}
}

const (
	OK_STATUS    = "OK"
	ERROR_STATUS = "Error"
)

func (publisher *PublisherType) GetServerName(ip string) string {
	// in case the IP is localhost, return current agent name
	islocal, err := IsLoopback(ip)
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

func (publisher *PublisherType) PublishEvent(ts time.Time, src *Endpoint, dst *Endpoint, event *outputs.Event) error {

	event.Src_server = publisher.GetServerName(src.Ip)
	event.Dst_server = publisher.GetServerName(dst.Ip)

	if _Config.Agent.Ignore_outgoing && event.Dst_server != "" &&
		event.Dst_server != publisher.name {
		// duplicated transaction -> ignore it
		logp.Debug("publish", "Ignore duplicated REDIS transaction on %s: %s -> %s", publisher.name, event.Src_server, event.Dst_server)
		return nil
	}

	event.Timestamp = ts
	event.Agent = publisher.name
	event.Src_ip = src.Ip
	event.Src_port = src.Port
	event.Src_proc = src.Proc
	event.Dst_ip = dst.Ip
	event.Dst_port = dst.Port
	event.Dst_proc = dst.Proc
	event.Tags = publisher.tags

	event.Src_country = ""
	if _GeoLite != nil {
		if len(event.Real_ip) > 0 {
			loc := _GeoLite.GetLocationByIP(event.Real_ip)
			if loc != nil {
				event.Src_country = loc.CountryCode
			}
		} else {
			// set src_country if no src_server is set
			if len(event.Src_server) == 0 { // only for external IP addresses
				loc := _GeoLite.GetLocationByIP(src.Ip)
				if loc != nil {
					event.Src_country = loc.CountryCode
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
			err := publisher.Output[i].PublishEvent(event)
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
		addrs, err := LocalIpAddrsAsStrings(false)
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

func (publisher *PublisherType) Init(publishDisabled bool) error {
	var err error

	publisher.disabled = publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	output, exists := _Config.Output["elasticsearch"]
	if exists && output.Enabled && !publisher.disabled {
		err := publisher.ElasticsearchOutput.Init(output,
			_Config.Agent.Topology_expire)
		if err != nil {
			logp.Err("Fail to initialize Elasticsearch as output: %s", err)
			return err
		}
		publisher.Output = append(publisher.Output, outputs.OutputInterface(&publisher.ElasticsearchOutput))

		if output.Save_topology {
			if publisher.TopologyOutput != nil {
				logp.Err("Multiple outputs defined to store topology. Please add save_topology = true option only for one output.")
				return errors.New("Multiple outputs defined to store topology")
			}
			publisher.TopologyOutput = outputs.OutputInterface(&publisher.ElasticsearchOutput)
			logp.Info("Using Elasticsearch to store the topology")
		}
	}

	output, exists = _Config.Output["redis"]
	if exists && output.Enabled && !publisher.disabled {
		logp.Debug("publish", "REDIS publisher enabled")
		err := publisher.RedisOutput.Init(output,
			_Config.Agent.Topology_expire)
		if err != nil {
			logp.Err("Fail to initialize Redis as output: %s", err)
			return err
		}
		publisher.Output = append(publisher.Output, outputs.OutputInterface(&publisher.RedisOutput))

		if output.Save_topology {
			if publisher.TopologyOutput != nil {
				logp.Err("Multiple outputs defined to store topology. Please add save_topology = true option only for one output.")
				return errors.New("Multiple outputs defined to store topology")
			}
			publisher.TopologyOutput = outputs.OutputInterface(&publisher.RedisOutput)
			logp.Info("Using Redis to store the topology")
		}
	}

	output, exists = _Config.Output["file"]
	if exists && output.Enabled {
		err := publisher.FileOutput.Init(output)
		if err != nil {
			logp.Err("Fail to initialize file output: %s", err)
			return err
		}
		publisher.Output = append(publisher.Output, outputs.OutputInterface(&publisher.FileOutput))

		// topology saving not supported by this one
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

	publisher.name = _Config.Agent.Name
	if len(publisher.name) == 0 {
		// use the hostname
		publisher.name, err = os.Hostname()
		if err != nil {
			return err
		}

		logp.Info("No agent name configured, using hostname '%s'", publisher.name)
	}

	if len(_Config.Agent.Tags) > 0 {
		publisher.tags = strings.Join(_Config.Agent.Tags, " ")
	}

	if !publisher.disabled && publisher.TopologyOutput != nil {
		RefreshTopologyFreq := 10 * time.Second
		if _Config.Agent.Refresh_topology_freq != 0 {
			RefreshTopologyFreq = time.Duration(_Config.Agent.Refresh_topology_freq) * time.Second
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

	return nil
}
