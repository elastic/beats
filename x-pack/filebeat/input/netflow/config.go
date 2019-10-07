// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/inputsource/udp"
)

type config struct {
	udp.Config                `config:",inline"`
	harvester.ForwarderConfig `config:",inline"`
	Protocols                 []string      `config:"protocols"`
	ExpirationTimeout         time.Duration `config:"expiration_timeout"`
	PacketQueueSize           int           `config:"queue_size"`
	CustomDefinitions         []string      `config:"custom_definitions"`
}

var defaultConfig = config{
	Config: udp.Config{
		MaxMessageSize: 10 * humanize.KiByte,
		Host:           ":2055",
		Timeout:        time.Minute * 5,
	},
	ForwarderConfig: harvester.ForwarderConfig{
		Type: inputName,
	},
	Protocols:         []string{"v5", "v9", "ipfix"},
	ExpirationTimeout: time.Minute * 30,
	PacketQueueSize:   8192,
}
