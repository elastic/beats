package cmd

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
)

func filebeatCfg(rawIn management.UnitsConfig) ([]*reload.ConfigWithMeta, error) {
	management.InjectStreamProcessor(&rawIn, "logs")
	management.InjectIndexProcessor(&rawIn, "logs")
	translateFilebeatType(&rawIn)

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromStreams(rawIn)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}
	return configList, nil
}

func translateFilebeatType(rawIn *management.UnitsConfig) {
	// I'm not sure what this does
	if rawIn.UnitType == "logfile" || rawIn.UnitType == "event/file" {
		rawIn.UnitType = "log"
	} else if rawIn.UnitType == "event/stdin" {
		rawIn.UnitType = "stdin"
	} else if rawIn.UnitType == "event/tcp" {
		rawIn.UnitType = "tcp"
	} else if rawIn.UnitType == "event/udp" {
		rawIn.UnitType = "udp"
	} else if rawIn.UnitType == "log/docker" {
		rawIn.UnitType = "docker"
	} else if rawIn.UnitType == "log/redis_slowlog" {
		rawIn.UnitType = "redis"
	} else if rawIn.UnitType == "log/syslog" {
		rawIn.UnitType = "syslog"
	}
}
