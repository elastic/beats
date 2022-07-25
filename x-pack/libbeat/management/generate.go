package management

import (
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/reload"
)

// UnitsConfig is an attempt at standardizing the config that beats will get via the V2
// See the related gdoc proposal for more information. For now, this is a tad sketchy,
// as the whole fleet stack is distributed enough that we can't be 100% sure this
// struct will be valid for long, or if we should expect unexpected data.
type UnitsConfig struct {
	Name       string     `yaml:"name"`
	ID         string     `yaml:"id"`
	UnitType   string     `yaml:"type"`
	Revision   int        `yaml:"revision"`
	UseOutput  string     `yaml:"use_output"`
	Meta       Meta       `yaml:"meta"`
	DataStream DataStream `yaml:"data_stream"`
	// For now, Streams has to stay in raw form, since the unit-level and agent-level fields aren't really namespaced
	Streams []string `yaml:"streams"`
}

// Meta is for fleet input metadata
type Meta struct {
	Package Package `yaml:"package"`
}

// Package is for package-related input metadata
type Package struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type DataStream struct {
	Dataset    string `yaml:"dataset"`
	StreamType string `yaml:"type"`
	Namespace  string `yaml:"namespace"`
}

type BeatCfgProcessor func(UnitsConfig) (*reload.ConfigWithMeta, error)

// mapping of input types to what beat we're on
// This is (hopefully) a temporary hack while everything is in-flight
// In later versions it would be nice to import whatever specs get loaded via the elastic-agent libs
// Skip this for metricbeat, since it's the only thing that registers */metrics
var typeMap = map[string]BeatCfgProcessor{}

// This generates an opaque config blob used by all the beats
// This has to handle both universal config changes and changes specific to the beats
func generateBeatConfig(rawIn UnitsConfig) (*reload.ConfigWithMeta, error) {

	//FixStreamRule
	if rawIn.DataStream.Namespace == "" {
		rawIn.DataStream.Namespace = "default"
	}
	if rawIn.DataStream.Dataset == "" {
		rawIn.DataStream.Dataset = "generic"
	}

	// InjectAgentInfoRule

	// In the AST, this rule will try to do something like this:
	/*
		"add_fields": {
			"fields": {
			"id": "521542dc-2369-4cd0-9f04-4c89e2603238",
			"snapshot": false,
			"version": "8.4.0"
			},
			"target": "elastic_agent"
		}
		"add_fields": {
				"fields": {
					"id": "521542dc-2369-4cd0-9f04-4c89e2603238"
				},
				"target": "agent"
		}
	*/
	// This requires an AgentInfo Struct that I don't seem to have access to.
	// Ditto for InjectHeadersRule

	// sort the config object to the applicable beat
	var metaConfig *reload.ConfigWithMeta
	var err error
	if strings.Contains(rawIn.UnitType, "/metrics") {
		metaConfig, err = metricbeatCfg(rawIn)

	}

	return metaConfig, err
}

func metricbeatCfg(rawIn UnitsConfig) (*reload.ConfigWithMeta, error) {

}
