package cmd

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func filebeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	modules, err := management.CreateInputsFromStreams(rawIn, "logs", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %s", err)
	}
	for iter := range modules {
		modules[iter]["type"] = translateFilebeatType(rawIn)
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}
	return configList, nil
}

func translateFilebeatType(rawIn *proto.UnitExpectedConfig) string {
	// I'm not sure what this does
	if rawIn.Type == "logfile" || rawIn.Type == "event/file" {
		return "log"
	} else if rawIn.Type == "event/stdin" {
		return "stdin"
	} else if rawIn.Type == "event/tcp" {
		return "tcp"
	} else if rawIn.Type == "event/udp" {
		return "udp"
	} else if rawIn.Type == "log/docker" {
		return "docker"
	} else if rawIn.Type == "log/redis_slowlog" {
		return "redis"
	} else if rawIn.Type == "log/syslog" {
		return "syslog"
	}
	return rawIn.Type
}
