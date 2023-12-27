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
func (r *TransformRegister) Transform(
	cfg *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
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
// Public config transformation
// ===========

// handleSimpleConfig perform the necessary checks and transformations in the raw
// config for the simple configuration case (no `streams` key in the config).
//
// Currently only the the `filestream` input is supported for the simple config.
func handleSimpleConfig(raw *proto.UnitExpectedConfig) (map[string]any, error) {
	m := raw.Source.AsMap()
	kind, ok := m["type"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for 'type', got %T", m["type"])
	}
	if kind != "filestream" {
		return nil, fmt.Errorf("the only supported type is 'filestream', got: %q", kind)
	}

	return m, nil
}

// CreateInputsFromStreams breaks down the raw Expected config into an array of individual inputs/modules from the Streams values
// that can later be formatted into the reloader's ConfigWithMetaData and sent to an indvidual beat/
// This also performs the basic task of inserting module-level add_field processors into the inputs/modules.
func CreateInputsFromStreams(raw *proto.UnitExpectedConfig, defaultDataStreamType string, agentInfo *client.AgentInfo, defaultProcessors ...mapstr.M) ([]map[string]interface{}, error) {
	// If there are no streams, we fall into the 'simple input config' case,
	// this means the key configuration values are on the root level instead of
	// an element in the `streams` array.
	if raw.GetStreams() == nil {
		streamSource, err := handleSimpleConfig(raw)
		if err != nil {
			return []map[string]interface{}{}, err
		}

		// Create stream rules with all the defaults and an empty streams struct.
		streamSource, err = createStreamRules(raw, streamSource, &proto.Stream{}, defaultDataStreamType, agentInfo, defaultProcessors...)
		if err != nil {
			return nil, fmt.Errorf("error creating stream rules for a simple config (empty streams array): %w", err)
		}

		return []map[string]interface{}{streamSource}, nil
	}
	inputs := make([]map[string]interface{}, len(raw.GetStreams()))

	for iter, stream := range raw.GetStreams() {
		streamSource := raw.GetStreams()[iter].GetSource().AsMap()
		streamSource, err := createStreamRules(raw, streamSource, stream, defaultDataStreamType, agentInfo, defaultProcessors...)
		if err != nil {
			return nil, fmt.Errorf("error creating stream rules: %w", err)
		}

		inputs[iter] = streamSource
	}

	return inputs, nil
}

// CreateShipperInput is a modified version of CreateInputsFromStreams made for forwarding input units to the shipper beat
// this does not create separate inputs for each stream, and instead passes it along as a single input, with just the processors added
func CreateShipperInput(raw *proto.UnitExpectedConfig, defaultDataStreamType string, agentInfo *client.AgentInfo, defaultProcessors ...mapstr.M) ([]map[string]interface{}, error) {
	inputs := make([]map[string]interface{}, len(raw.GetStreams()))
	for iter, stream := range raw.GetStreams() {
		streamSource := raw.GetStreams()[iter].GetSource().AsMap()
		streamSource = injectIndexStream(defaultDataStreamType, raw, stream, streamSource)
		// 1. global processors
		streamSource = injectGlobalProcesssors(raw, streamSource)

		// 2. agentInfo
		streamSource, err := injectAgentInfoRule(streamSource, agentInfo)
		if err != nil {
			return nil, fmt.Errorf("Error injecting agent processors: %w", err)
		}

		// 3. stream processors
		streamSource, err = injectStreamProcessors(raw, defaultDataStreamType, stream, streamSource, defaultProcessors)
		if err != nil {
			return nil, fmt.Errorf("Error injecting stream processors: %w", err)
		}
		inputs[iter] = streamSource
	}
	rawMap := raw.Source.AsMap()
	rawMap["streams"] = inputs

	return []map[string]interface{}{rawMap}, nil
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

// ===========
// config injection
// ===========

// convinence method for wrapping all the stream transformations needed by the shipper and other inputs
func createStreamRules(raw *proto.UnitExpectedConfig, streamSource map[string]interface{}, stream *proto.Stream, defaultDataStreamType string, agentInfo *client.AgentInfo, defaultProcessors ...mapstr.M) (map[string]interface{}, error) {

	streamSource = injectIndexStream(defaultDataStreamType, raw, stream, streamSource)

	// the order of building the processors is important
	// prepend is used to ensure that the processors defined directly on the stream
	// come last in the order as they take priority over the others as they are the
	// most specific to this one stream

	// 1. global processors
	streamSource = injectGlobalProcesssors(raw, streamSource)

	// 2. agentInfo
	streamSource, err := injectAgentInfoRule(streamSource, agentInfo)
	if err != nil {
		return nil, fmt.Errorf("Error injecting agent processors: %w", err)
	}

	// 3. stream processors
	streamSource, err = injectStreamProcessors(raw, defaultDataStreamType, stream, streamSource, defaultProcessors)
	if err != nil {
		return nil, fmt.Errorf("Error injecting stream processors: %w", err)
	}

	// now the order of the processors on this input is as follows
	// 1. stream processors
	// 2. agentInfo processors
	// 3. global processors
	// 4. stream specific processors

	return streamSource, nil
}

// Emulates the InjectAgentInfoRule and InjectHeadersRule ast rules
// adds processors for agent-related metadata
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

	inputs["processors"] = prependProcessors(inputs, processors)

	return inputs, nil
}

// injectGlobalProcesssors re-injects any global processors into the individual streams
func injectGlobalProcesssors(expected *proto.UnitExpectedConfig, stream map[string]interface{}) map[string]interface{} {
	rootMap := expected.GetSource().AsMap()
	globalProcFound, ok := rootMap["processors"]
	if !ok {
		return stream
	}
	globalList, ok := globalProcFound.([]interface{})
	if !ok {
		return stream
	}
	// copy global processors to ensure that each stream gets its own copy
	// if the stream doesn't have any processors it will take the slice as the new value
	// without copying its possible that the processors appended to the streams will be shared
	newProcs := prependProcessors(stream, append([]interface{}{}, globalList...))
	stream["processors"] = newProcs
	return stream
}

// injectIndexStream is an emulation of the InjectIndexProcessor AST code
// this adds the `index` field, based on the data_stream info we get from the config
func injectIndexStream(defaultDataStreamType string, expected *proto.UnitExpectedConfig, streamExpected *proto.Stream, stream map[string]interface{}) map[string]interface{} {
	streamType, dataset, namespace := metadataFromDatastreamValues(defaultDataStreamType, expected, streamExpected)
	index := fmt.Sprintf("%s-%s-%s", streamType, dataset, namespace)
	stream["index"] = index
	return stream
}

// injectStreamProcessors is an emulation of the InjectStreamProcessorRule AST code
// this adds a variety of processors for metadata related to the dataset and input config.
func injectStreamProcessors(expected *proto.UnitExpectedConfig, defaultDataStreamType string, streamExpected *proto.Stream, stream map[string]interface{}, defaultProcessors []mapstr.M) (map[string]interface{}, error) {
	// 1. start by "repairing" config to add any missing fields
	// logic from datastreamTypeFromInputNode
	procInputType, procInputDataset, procInputNamespace := metadataFromDatastreamValues(defaultDataStreamType, expected, streamExpected)

	var processors = []interface{}{}

	for _, p := range defaultProcessors {
		if len(p) == 0 {
			continue
		}
		processors = append(processors, p)
	}

	// In V1, AST injects input_id at the input level and not the stream level,
	// for reasons I can't understand, as it just ends up shuffling it around
	// to individual metricsets anyway, at least on metricbeat
	if expectedID := expected.GetId(); expectedID != "" {
		inputID := generateAddFieldsProcessor(mapstr.M{"input_id": expectedID}, "@metadata")
		processors = append(processors, inputID)
	}

	// 2. Actually add the processors
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

	// prepend with existing processors
	// streams processors should be first as other processors might adjust values in those fields
	stream["processors"] = prependProcessors(stream, processors)

	return stream, nil
}

// ===========
// Config Processors
// ===========

// This generates an opaque config blob used by all the beats
// This has to handle both universal config changes and changes specific to the beats
// This is a replacement for the AST code that lived in V1
func generateBeatConfig(unitRaw *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {

	// Generate the config that's unique to a beat
	metaConfig, err := ConfigTransform.Transform(unitRaw, agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error transforming config for beats: %w", err)
	}
	return metaConfig, nil
}

// ===========
// helpers
// ===========

func generateAddFieldsProcessor(fields mapstr.M, target string) mapstr.M {
	return mapstr.M{
		"add_fields": mapstr.M{
			"fields": fields,
			"target": target,
		},
	}
}

// prependProcessors takes an existing input or stream-level config, extracts any existing processors in the config,
// and appends them to a new list of configs. Mostly a helper to deal with all the typecasting
func prependProcessors(existingConfig map[string]interface{}, newProcs []interface{}) []interface{} {
	currentProcs, ok := existingConfig["processors"]
	if !ok {
		return newProcs
	}
	currentList, ok := currentProcs.([]interface{})
	if !ok {
		return newProcs
	}
	return append(newProcs, currentList...)
}

// metadataFromDatastreamValues takes the various data_stream values from across the expected config and returns a set of "good" that can be used to add fields
// to the final beat config. The underlying logic follows a series of steps:
// 1) start with a set of default values, including the specified defaultDataStream
// 2) set the values based on the data_stream fields at the input level (i.e, the UnitExpectedConfig)
// 3) set the values based on the data_stream fields at the stream level (i.e., the proto.Stream)
// This returns: type, dataset, namespace
func metadataFromDatastreamValues(defaultDataStreamType string, expected *proto.UnitExpectedConfig, streamExpected *proto.Stream) (string, string, string) {
	// type
	setType := defaultDataStreamType
	if newType := streamExpected.GetDataStream().GetType(); newType != "" {
		setType = newType
	}
	// if we get a unit-level value, but it's the default, don't overwrite
	if newType := expected.GetDataStream().GetType(); newType != "" && newType != defaultDataStreamType {
		setType = newType
	}

	// dataset
	setDataset := DefaultDatasetName
	if newDataset := streamExpected.GetDataStream().GetDataset(); newDataset != "" {
		setDataset = newDataset
	}
	if newDataset := expected.GetDataStream().GetDataset(); newDataset != "" && newDataset != DefaultDatasetName {
		setDataset = newDataset
	}

	// namespace
	setNamespace := DefaultNamespaceName
	if newNamespace := streamExpected.GetDataStream().GetNamespace(); newNamespace != "" {
		setNamespace = newNamespace
	}
	if newNamespace := expected.GetDataStream().GetNamespace(); newNamespace != "" && newNamespace != DefaultDatasetName {
		setNamespace = newNamespace
	}

	return setType, setDataset, setNamespace
}

// generate the output config, including shuffling around the `type` key
// In V1, this was done by the groupByOutputs function buried in the AST init
func groupByOutputs(outCfg *proto.UnitExpectedConfig) (*reload.ConfigWithMeta, error) {
	// We still need to emulate the InjectHeadersRule AST code,
	// I don't think we can get the `Headers()` data reported by the AgentInfo()
	sourceMap := outCfg.GetSource().AsMap()
	outputType := outCfg.GetType() //nolint:typecheck // this is used, linter just doesn't seem to see it
	if outputType == "" {
		return nil, fmt.Errorf("output config does not have a configured type field")
	}
	formattedOut := mapstr.M{
		outputType: sourceMap,
	}
	uconfig, err := conf.NewConfigFrom(formattedOut)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config for output: %w", err)
	}

	return &reload.ConfigWithMeta{Config: uconfig}, nil
}
