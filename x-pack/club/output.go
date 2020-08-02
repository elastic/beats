package main

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"

	_ "github.com/elastic/beats/v7/libbeat/publisher/includes"
)

type outputConfig struct {
	Output   common.ConfigNamespace `config:"output"`
	Pipeline pipeline.Config        `config:",inline"`
}

type outputManager struct {
	pipelines map[string]*pipeline.Pipeline
}

func configureOutputs(
	log *logp.Logger,
	info beat.Info,
	outputConfig map[string]*common.Config,
	rawConfig *common.Config,
) (*outputManager, error) {
	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, info, rawConfig)
	if err != nil {
		return nil, err
	}

	pipelines := map[string]*pipeline.Pipeline{}
	for name, cfg := range outputConfig {
		typeInfo := struct{ Type string }{}
		if err := cfg.Unpack(&typeInfo); err != nil {
			return nil, err
		}

		var pipeConfig pipeline.Config
		if err := cfg.Unpack(&pipeConfig); err != nil {
			return nil, err
		}

		pipeline, err := pipeline.Load(info,
			pipeline.Monitors{
				Metrics:   nil,
				Telemetry: nil,
				Logger:    log.Named("publisher"),
				Tracer:    nil,
			},
			pipeConfig,
			nil,
			makeOutputFactory(info, indexManagement, typeInfo.Type, cfg),
		)
		if err != nil {
			return nil, err
		}
		pipelines[name] = pipeline
	}

	return &outputManager{pipelines: pipelines}, nil
}

func makeOutputFactory(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	outputType string,
	cfg *common.Config,
) func(outputs.Observer) (string, outputs.Group, error) {
	return func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := outputs.Load(indexManagement, info, outStats, outputType, cfg)
		return outputType, out, err
	}
}

func (m *outputManager) Close() {
	for _, pipeline := range m.pipelines {
		pipeline.Close()
	}
}

func (m *outputManager) GetPipeline(name string) beat.Pipeline {
	return m.pipelines[name]
}
