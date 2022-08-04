package management

import (
	"fmt"
	"os"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"gopkg.in/yaml.v2"
)

// TransformRegister is a hack that allows an individual beat to set a transform function
// so the V2 controller can perform beat-specific config transformations.
// This is mostly done this way so we can avoid mixing up code with different licenses,
// as this is entirely xpack/Elastic License code, and the entire beats setup process happens in libbeat/
type TransformRegister struct {
	transformFunc func(UnitsConfig) ([]*reload.ConfigWithMeta, error)
}

// SetTransform sets a transform function callback
func (r *TransformRegister) SetTransform(func(UnitsConfig) ([]*reload.ConfigWithMeta, error)) {

}

// SetTransform sets a transform function callback
func (r *TransformRegister) Transform(UnitsConfig) ([]*reload.ConfigWithMeta, error) {
	return nil, nil
}

// StreamConfig is a wrapper type so we can correct the behavior of yaml.Unmarshal, see UnmarshalYAML() below
type StreamConfig map[string]interface{}

// Figure out what beat we're running as.
// There's almost certainly a better way to do this,
// but for now it works
var exeName = os.Args[0]

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
	Streams []StreamConfig `yaml:"streams"`
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
	Dataset    string `yaml:"dataset" struct:"dataset"`
	StreamType string `yaml:"type" struct:"type"`
	Namespace  string `yaml:"namespace" struct:"namespace"`
}

// StreamMetadata is a helper for unmarshalling stream-level config
// while we do transformations to generate processors
type StreamMetadata struct {
	DataStream DataStream `yaml:"data_stream" struct:"data_stream"`
	Processors []mapstr.M `yaml:"processors" struct:"processors"`
}

// UnmarshalYAML is a little helper to deal with the fact that the yaml unmarshaler will create
// hashmaps of type map[interface{}]interface{}, which breaks a bunch of stuff.
// This code was actually taken from an old version of libbeat circa 2015, which was removed for reasons I don't understand.
func (st *StreamConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var result map[interface{}]interface{}
	err := unmarshal(&result)
	if err != nil {
		return fmt.Errorf("Error in custom unmarshalYAML: %w", err)
	}
	*st = cleanUpInterfaceMap(result)
	return nil
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) StreamConfig {
	result := make(StreamConfig)
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

func generateAddFieldsProcessor(fields mapstr.M, target string) mapstr.M {
	return mapstr.M{
		"add_fields": mapstr.M{
			"fields": fields,
			"target": target,
		},
	}
}

// This generates an opaque config blob used by all the beats
// This has to handle both universal config changes and changes specific to the beats
// This is a replacement for the AST code that lived in V1
func generateBeatConfig(unitRaw string) ([]*reload.ConfigWithMeta, error) {
	rawIn := UnitsConfig{}

	err := yaml.Unmarshal([]byte(unitRaw), &rawIn)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling unit config: %w", err)
	}
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
	var metaConfig []*reload.ConfigWithMeta
	if strings.Contains(exeName, "metricbeat") {
		metaConfig, err = metricbeatCfg(rawIn)
	} else if strings.Contains(exeName, "filebeat") {
		metaConfig, err = filebeatCfg(rawIn)
	}

	return metaConfig, err
}

// generate the output config, including shuffling around the `type` key
// In V1, this was done by the groupByOutputs function buried in the AST init
func groupByOutputs(outCfg string) (*reload.ConfigWithMeta, error) {
	outMap := mapstr.M{}
	err := yaml.Unmarshal([]byte(outCfg), &outMap)
	if err != nil {
		return nil, err
	}

	outputType := ""
	for cfgk, val := range outMap {
		if cfgk == "type" {
			outputType = val.(string)
		}
	}
	formattedOut := mapstr.M{
		outputType: outMap,
	}

	uconfig, err := conf.NewConfigFrom(formattedOut)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config for output: %w", err)
	}
	return &reload.ConfigWithMeta{Config: uconfig}, nil
}

// InjectIndexProcessor is an emulation of the InjectIndexProcessor AST code
func InjectIndexProcessor(rawIn *UnitsConfig, inputType string) {

	for iter := range rawIn.Streams {
		var streamDS StreamMetadata
		streamCfg := rawIn.Streams[iter]

		err := typeconv.Convert(&streamDS, streamCfg)
		if err != nil {
			fmt.Printf("Error fetching stream-level datastream: %s\n", err)
		}

		streamType := streamDS.DataStream.StreamType
		if streamType == "" {
			streamType = inputType
		}
		dataset := streamDS.DataStream.Dataset
		if dataset == "" {
			dataset = "generic"
		}
		namespace := rawIn.DataStream.Namespace
		if namespace == "" {
			namespace = "default"
		}
		index := fmt.Sprintf("%s-%s-%s", streamType, dataset, namespace)
		rawIn.Streams[iter]["index"] = index

	}
}

//InjectStreamProcessor is an emulation of the InjectStreamProcessorRule AST code
func InjectStreamProcessor(rawIn *UnitsConfig, inputType string) {
	// logic from datastreamTypeFromInputNode
	procInputType := inputType
	if rawIn.DataStream.StreamType != "" {
		procInputType = rawIn.DataStream.StreamType
	}

	procInputNamespace := "default"
	if rawIn.DataStream.Namespace != "" {
		procInputNamespace = rawIn.DataStream.Namespace
	}

	for iter := range rawIn.Streams {
		var streamDS StreamMetadata
		streamCfg := rawIn.Streams[iter]

		err := typeconv.Convert(&streamDS, streamCfg)
		// TODO: set up some kind of logging here
		if err != nil {
			fmt.Printf("Error fetching stream-level datastream: %s\n", err)
		}

		var processors = []mapstr.M{}
		// the AST injects input_id at the input level and not the stream level,
		// for reasons I can't understand, as it just ends up shuffling it around
		// to individual metricsets anyway, at least on metricbeat
		inputId := generateAddFieldsProcessor(mapstr.M{"input_id": rawIn.ID}, "@metadata")
		processors = append(processors, inputId)

		procInputDataset := "generic"
		if streamDS.DataStream.Dataset != "" {
			procInputDataset = streamDS.DataStream.Dataset
		}

		// namespace
		datastream := generateAddFieldsProcessor(mapstr.M{"dataset": procInputDataset,
			"namespace": procInputNamespace, "type": procInputType}, "data_stream")
		processors = append(processors, datastream)

		// dataset
		event := generateAddFieldsProcessor(mapstr.M{"dataset": procInputDataset}, "event")
		processors = append(processors, event)

		// source stream
		streamID, ok := streamCfg["id"]
		if ok {
			sourceStream := generateAddFieldsProcessor(mapstr.M{"stream_id": streamID}, "@metadata")
			processors = append(processors, sourceStream)
		}

		var updatedProcs = []mapstr.M{}
		// append to the existing processor list, if it exists
		if len(streamDS.Processors) == 0 {
			updatedProcs = processors
		} else {
			updatedProcs = append(streamDS.Processors, processors...)
		}
		rawIn.Streams[iter]["processors"] = updatedProcs
	} // end of stream loop

}

// FormatMetricbeatModules is a combination of the map and rename rules in the metricbeat spec file,
// and formats various key values needed by metricbeat
func FormatMetricbeatModules(rawIn *UnitsConfig) {
	// Extract the module name from the type, usually in the form system/metric
	module := strings.Split(rawIn.UnitType, "/")[0]

	for iter := range rawIn.Streams {
		rawIn.Streams[iter]["module"] = module
	}

}

func translateFilebeatType(rawIn *UnitsConfig) {
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

func metricbeatCfg(rawIn UnitsConfig) ([]*reload.ConfigWithMeta, error) {

	InjectStreamProcessor(&rawIn, "metrics")
	InjectIndexProcessor(&rawIn, "metrics")
	FormatMetricbeatModules(&rawIn)

	// format for the reloadable list needed bythe cm.Reload() method
	configList := make([]*reload.ConfigWithMeta, len(rawIn.Streams))

	for iter := range rawIn.Streams {
		//cfg := mapstr.M{"modules": withProcessors.Streams[iter]}
		uconfig, err := conf.NewConfigFrom(rawIn.Streams[iter])
		if err != nil {
			return nil, fmt.Errorf("error in conversion to conf.C:")
		}
		configList[iter] = &reload.ConfigWithMeta{Config: uconfig}
	}

	return configList, nil
}

func filebeatCfg(rawIn UnitsConfig) ([]*reload.ConfigWithMeta, error) {
	InjectStreamProcessor(&rawIn, "logs")
	InjectIndexProcessor(&rawIn, "logs")
	translateFilebeatType(&rawIn)

	// format for the reloadable list needed bythe cm.Reload() method
	configList := make([]*reload.ConfigWithMeta, len(rawIn.Streams))
	for iter := range rawIn.Streams {
		uconfig, err := conf.NewConfigFrom(rawIn.Streams[iter])
		if err != nil {
			return nil, fmt.Errorf("error in conversion to conf.C:")
		}
		configList[iter] = &reload.ConfigWithMeta{Config: uconfig}
	}

	return configList, nil
}

// A little debug helper to print everything in a config once its been rendered
// TODO: turn this into a debug flag or something?
func printConfigDebug(cfg []*reload.ConfigWithMeta) string {
	stringAcc := ""
	for _, cfgItem := range cfg {
		cfgMap := mapstr.M{}
		err := cfgItem.Config.Unpack(&cfgMap)
		if err != nil {
			stringAcc = fmt.Sprintf("%s\n%s\n", stringAcc, err)
		}
		stringAcc = fmt.Sprintf("%s\n%s\n", stringAcc, cfgMap.StringToPrint())
	}
	return stringAcc
}
