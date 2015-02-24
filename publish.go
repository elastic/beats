package main

import (
	"encoding/json"
	"errors"
	"os"
	"packetbeat/common"
	"packetbeat/config"
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

type Topology struct {
	Name string `json:"name"`
	Ip   string `json:"ip"`
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

func (publisher *PublisherType) PublishEvent(ts time.Time, src *common.Endpoint, dst *common.Endpoint, event common.MapStr) error {

	src_server := publisher.GetServerName(src.Ip)
	dst_server := publisher.GetServerName(dst.Ip)

	if config.ConfigSingleton.Agent.Ignore_outgoing && dst_server != "" &&
		dst_server != publisher.name {
		// duplicated transaction -> ignore it
		logp.Debug("publish", "Ignore duplicated transaction on %s: %s -> %s", publisher.name, src_server, dst_server)
		return nil
	}
	event["client_server"] = src_server
	event["server"] = dst_server

	event["timestamp"] = ts
	event["agent"] = publisher.name
	event["client_ip"] = src.Ip
	event["client_port"] = src.Port
	event["client_proc"] = src.Proc
	event["ip"] = dst.Ip
	event["port"] = dst.Port
	event["proc"] = dst.Proc
	event["tags"] = publisher.tags

	event["country"] = ""
	if _GeoLite != nil {
		real_ip, exists := event["real_ip"]
		if exists && len(real_ip.(string)) > 0 {
			loc := _GeoLite.GetLocationByIP(real_ip.(string))
			if loc != nil {
				event["country"] = loc.CountryCode
			}
		} else {
			if len(src_server) == 0 { // only for external IP addresses
				loc := _GeoLite.GetLocationByIP(src.Ip)
				if loc != nil {
					event["country"] = loc.CountryCode
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
			err := publisher.Output[i].PublishEvent(ts, event)
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

func (publisher *PublisherType) Init(publishDisabled bool) error {
	var err error

	publisher.disabled = publishDisabled
	if publisher.disabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	output, exists := config.ConfigSingleton.Output["elasticsearch"]
	if exists && output.Enabled && !publisher.disabled {
		err := publisher.ElasticsearchOutput.Init(output,
			config.ConfigSingleton.Agent.Topology_expire)
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

	output, exists = config.ConfigSingleton.Output["redis"]
	if exists && output.Enabled && !publisher.disabled {
		logp.Debug("publish", "REDIS publisher enabled")
		err := publisher.RedisOutput.Init(output,
			config.ConfigSingleton.Agent.Topology_expire)
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

	output, exists = config.ConfigSingleton.Output["file"]
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

	publisher.name = config.ConfigSingleton.Agent.Name
	if len(publisher.name) == 0 {
		// use the hostname
		publisher.name, err = os.Hostname()
		if err != nil {
			return err
		}

		logp.Info("No agent name configured, using hostname '%s'", publisher.name)
	}

	if len(config.ConfigSingleton.Agent.Tags) > 0 {
		publisher.tags = strings.Join(config.ConfigSingleton.Agent.Tags, " ")
	}

	if !publisher.disabled && publisher.TopologyOutput != nil {
		RefreshTopologyFreq := 10 * time.Second
		if config.ConfigSingleton.Agent.Refresh_topology_freq != 0 {
			RefreshTopologyFreq = time.Duration(config.ConfigSingleton.Agent.Refresh_topology_freq) * time.Second
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
