// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// DefaultNamespaceName is the fallback default namespace for data stream info
var DefaultNamespaceName = "default"

// DefaultDatasetName is the fallback default dataset for data stream info
var DefaultDatasetName = "generic"

// ===========
// Config Transformation Registry
// ===========

// ConfigTransform is the global registry value for beat's config transformation callback
var ConfigTransform = TransformRegister{}

// TransformRegister is a hack that allows an individual beat to set a transform function
// so the V2 controller can perform beat-specific config transformations.
// This is mostly done this way so we can avoid mixing up code with different licenses,
// as this is entirely xpack/Elastic License code, and the normal beat init process happens in libbeat.
// This is fairly simple, as only one beat will ever register a callback.
type TransformRegister struct {
	transformFunc func(*proto.UnitExpectedConfig, *client.AgentInfo) ([]*reload.ConfigWithMeta, error)
}

// SetTransform sets a transform function callback
func (r *TransformRegister) SetTransform(transform func(*proto.UnitExpectedConfig, *client.AgentInfo) ([]*reload.ConfigWithMeta, error)) {
	r.transformFunc = transform
}

// Transform sets a transform function callback
func (r *TransformRegister) Transform(cfg *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	// If no transform is registered, fallback to a basic setup
	if r.transformFunc == nil {
		streamList, err := CreateInputsFromStreams(cfg, "log", agentInfo)
		if err != nil {
			return nil, fmt.Errorf("error creating input list from fallback function: %w", err)
		}
		// format for the reloadable list needed bythe cm.Reload() method
		configList, err := CreateReloadConfigFromInputs(streamList)
		if err != nil {
			return nil, fmt.Errorf("error creating reloader config: %w", err)
		}
		return configList, nil
	}

	return r.transformFunc(cfg, agentInfo)
}

// ===========
// Stream and Input processors
// ===========

// CreateInputsFromStreams breaks down the raw Expected config into an array of individual inputs/modules from the Streams values
// that can later be formatted into the reloader's ConfigWithMetaData and sent to an indvidual beat/
// This also performs the basic task of inserting module-level add_field processors into the inputs/modules.
func CreateInputsFromStreams(raw *proto.UnitExpectedConfig, inputType string, agentInfo *client.AgentInfo) ([]map[string]interface{}, error) {
	inputs := make([]map[string]interface{}, len(raw.Streams))

	for iter, stream := range raw.GetStreams() {
		streamSource := raw.GetStreams()[iter].GetSource().AsMap()

		streamSource = injectIndexStream(inputType, stream, streamSource)
		streamSource, err := injectStreamProcessors(raw, inputType, stream, streamSource)
		if err != nil {
			return nil, fmt.Errorf("Error injecting stream processors: %w", err)
		}
		streamSource, err = injectAgentInfoRule(streamSource, agentInfo)
		if err != nil {
			return nil, fmt.Errorf("Error injecting agent processors: %w", err)
		}
		inputs[iter] = streamSource
	}

	return inputs, nil
}

// CreateReloadConfigFromInputs turns a raw input/module list into the ConfigWithMeta type used by the reloader interface
func CreateReloadConfigFromInputs(raw []map[string]interface{}) ([]*reload.ConfigWithMeta, error) {
	// format for the reloadable list needed bythe cm.Reload() method
	configList := make([]*reload.ConfigWithMeta, len(raw))

	for iter := range raw {
		uconfig, err := conf.NewConfigFrom(raw[iter])
		if err != nil {
			return nil, fmt.Errorf("error in conversion to conf.C: %w", err)
		}
		configList[iter] = &reload.ConfigWithMeta{Config: uconfig}
	}
	return configList, nil
}

// Emulates the InjectAgentInfoRule and InjectHeadersRule ast rules
func injectAgentInfoRule(inputs map[string]interface{}, agentInfo *client.AgentInfo) (map[string]interface{}, error) {
	// upstream API can sometimes return a nil agent info
	if agentInfo == nil {
		return inputs, nil
	}
	var processors []interface{}

	processors = append(processors, generateAddFieldsProcessor(
		mapstr.M{"id": agentInfo.ID, "snapshot": agentInfo.Snapshot, "version": agentInfo.Version},
		"elastic_agent"))
	processors = append(processors, generateAddFieldsProcessor(
		mapstr.M{"id": agentInfo.ID},
		"agent"))

	currentProcs, ok := inputs["processors"]
	if !ok {
		inputs["processors"] = processors
	} else {
		currentProcsList, ok := currentProcs.([]interface{})
		if !ok {
			return nil, fmt.Errorf("error creating list of existing processors, got: %#v", currentProcs)
		}
		inputs["processors"] = append(processors, currentProcsList...)

	}

	return inputs, nil
}

// injectIndexStream is an emulation of the InjectIndexProcessor AST code
func injectIndexStream(inputType string, streamExpected *proto.Stream, stream map[string]interface{}) map[string]interface{} {
	streamType := streamExpected.GetDataStream().GetType()
	if streamType == "" {
		streamType = inputType
	}

	dataset := DefaultDatasetName
	if testDataset := streamExpected.GetDataStream().GetDataset(); testDataset != "" {
		dataset = testDataset
	}

	namespace := DefaultNamespaceName
	if testNamespace := streamExpected.GetDataStream().GetNamespace(); testNamespace != "" {
		namespace = testNamespace
	}

	index := fmt.Sprintf("%s-%s-%s", streamType, dataset, namespace)
	stream["index"] = index
	return stream
}

//injectStreamProcessors is an emulation of the InjectStreamProcessorRule AST code
func injectStreamProcessors(expected *proto.UnitExpectedConfig, inputType string, streamExpected *proto.Stream, stream map[string]interface{}) (map[string]interface{}, error) {
	//1. start by "repairing" config to add any missing fields
	// logic from datastreamTypeFromInputNode
	procInputType := inputType
	if testInputType := expected.GetDataStream().GetType(); testInputType != "" {
		procInputType = testInputType
	}

	procInputNamespace := DefaultNamespaceName
	if testInputNamespace := expected.GetDataStream().GetNamespace(); testInputNamespace != "" {
		procInputNamespace = testInputNamespace
	}

	var processors = []interface{}{}

	// the AST injects input_id at the input level and not the stream level,
	// for reasons I can't understand, as it just ends up shuffling it around
	// to individual metricsets anyway, at least on metricbeat
	if expectedID := expected.GetId(); expectedID != "" {
		inputID := generateAddFieldsProcessor(mapstr.M{"input_id": expectedID}, "@metadata")
		processors = append(processors, inputID)
	}

	procInputDataset := DefaultDatasetName
	if testStreamDataset := streamExpected.GetDataStream().GetDataset(); testStreamDataset != "" {
		procInputDataset = testStreamDataset
	}

	//2. Actually add the processors
	// namespace
	datastream := generateAddFieldsProcessor(mapstr.M{"dataset": procInputDataset,
		"namespace": procInputNamespace, "type": procInputType}, "data_stream")
	processors = append(processors, datastream)

	// dataset
	event := generateAddFieldsProcessor(mapstr.M{"dataset": procInputDataset}, "event")
	processors = append(processors, event)

	// source stream
	if streamID := streamExpected.GetId(); streamID != "" {
		sourceStream := generateAddFieldsProcessor(mapstr.M{"stream_id": streamID}, "@metadata")
		processors = append(processors, sourceStream)
	}

	// figure out if we have any existing processors
	currentProcs, ok := stream["processors"]
	if !ok {
		stream["processors"] = processors
	} else {
		currentProcsList, ok := currentProcs.([]interface{})
		if !ok {
			return nil, fmt.Errorf("error creating list of existing processors, got: %#v", currentProcs)
		}
		stream["processors"] = append(processors, currentProcsList...)

	}

	return stream, nil
}

// ===========
// Config Processors
// ===========

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
func generateBeatConfig(unitRaw *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	// We aren't guaranteed a DataStream field from the config
	if unitRaw.GetDataStream() == nil {
		unitRaw.DataStream = &proto.DataStream{
			Namespace: DefaultNamespaceName,
			Dataset:   DefaultDatasetName,
		}
	} else {
		if unitRaw.GetDataStream().GetNamespace() == "" {
			unitRaw.DataStream.Namespace = DefaultNamespaceName
		}
		if unitRaw.GetDataStream().GetDataset() == "" {
			unitRaw.DataStream.Dataset = DefaultDatasetName
		}
	}

	// Generate the config that's unique to a beat
	metaConfig, err := ConfigTransform.Transform(unitRaw, agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error transforming config for beats: %w", err)
	}
	return metaConfig, nil
}

// generate the output config, including shuffling around the `type` key
// In V1, this was done by the groupByOutputs function buried in the AST init
func groupByOutputs(outCfg *proto.UnitExpectedConfig) (*reload.ConfigWithMeta, error) {
	// We still need to emulate the InjectHeadersRule AST code,
	// I don't think we can get the `Headers()` data reported by the AgentInfo()
	sourceMap := outCfg.GetSource().AsMap()
	outputType := outCfg.GetType()
	formattedOut := mapstr.M{
		outputType: sourceMap,
	}
	uconfig, err := conf.NewConfigFrom(formattedOut)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config for output: %w", err)
	}

	return &reload.ConfigWithMeta{Config: uconfig}, nil
}
