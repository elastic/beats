package main

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/collector/internal/adapter/beatsout"
	"github.com/elastic/beats/v7/x-pack/collector/internal/publishing"
	"github.com/elastic/beats/v7/x-pack/collector/os/console/consoleout"
)

func outputPlugins(info beat.Info) []publishing.Plugin {
	beatsOutput := func(name string) publishing.Plugin {
		return publishing.Plugin{
			Name:       name,
			Stability:  feature.Stable,
			Deprecated: false,
			Configure:  beatsout.NewOutputFactory(info, name).ConfigureOutput,
		}
	}

	return []publishing.Plugin{
		consoleout.Plugin(info),
		beatsOutput("file"),
		beatsOutput("elasticsearch"),
		beatsOutput("logstash"),
		beatsOutput("kafka"),
		beatsOutput("redis"),
	}
}
