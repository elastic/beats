package main

import (
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/publisher"
)

type TopConfig struct {
	Period *int64
}

type ConfigSettings struct {
	Input   TopConfig
	Output  map[string]outputs.MothershipConfig
	Logging logp.Logging
	Shipper publisher.ShipperConfig
}

var Config ConfigSettings
