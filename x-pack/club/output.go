package main

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"

	_ "github.com/elastic/beats/v7/libbeat/publisher/includes"
)

type outputConfig struct {
	Output   common.ConfigNamespace `config:"output"`
	Pipeline pipeline.Config        `config:",inline"`
}

func configurePublishingPipeline(log *logp.Logger, info beat.Info, config outputConfig, rawConfig *common.Config) (beat.Pipeline, func(), error) {
	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, info, rawConfig)
	if err != nil {
		return nil, nil, err
	}

	pipeline, err := pipeline.Load(info,
		pipeline.Monitors{
			Metrics:   nil,
			Telemetry: monitoring.GetNamespace("state").GetRegistry(),
			Logger:    log.Named("publisher"),
			Tracer:    nil,
		},
		config.Pipeline,
		nil,
		makeOutputFactory(info, indexManagement, config.Output),
	)
	if err != nil {
		return nil, nil, err
	}

	return pipeline, func() { pipeline.Close() }, nil
}

func makeOutputFactory(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	cfg common.ConfigNamespace,
) func(outputs.Observer) (string, outputs.Group, error) {
	return func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := createOutput(info, indexManagement, outStats, cfg)
		return cfg.Name(), out, err
	}
}

func createOutput(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	stats outputs.Observer,
	cfg common.ConfigNamespace,
) (outputs.Group, error) {
	if !cfg.IsSet() {
		return outputs.Group{}, nil
	}
	return outputs.Load(indexManagement, info, stats, cfg.Name(), cfg.Config())
}
