package pipeline

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
)

func createPublishPipeline(log *logp.Logger, info beat.Info, cfg *common.Config) (*beatpipe.Pipeline, error) {
	var pipeConfig beatpipe.Config
	if err := cfg.Unpack(&pipeConfig); err != nil {
		return nil, err
	}

	typeInfo := struct{ Type string }{}
	if err := cfg.Unpack(&typeInfo); err != nil {
		return nil, err
	}

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagementConfig := common.MustNewConfigFrom(map[string]interface{}{
		"setup.ilm.enabled":      false,
		"setup.template.enabled": false,
		"output.something":       map[string]interface{}{},
	})

	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, info, indexManagementConfig)
	if err != nil {
		// the config is hard coded, if we panic here, we've messed up
		panic(err)
	}

	outputPipeline, err := beatpipe.Load(info,
		beatpipe.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log.Named("publish"),
			Tracer:    nil,
		},
		pipeConfig,
		nil,
		makeOutputFactory(info, indexManagement, typeInfo.Type, cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid output configuration: %w", err)
	}

	return outputPipeline, nil
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
