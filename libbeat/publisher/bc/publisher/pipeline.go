package publisher

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/broker"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

func createPipeline(
	beatInfo common.BeatInfo,
	shipper ShipperConfig,
	processors *processors.Processors,
	outcfg common.ConfigNamespace,
) (*pipeline.Pipeline, error) {

	reg := monitoring.Default.GetRegistry("libbeat")
	if reg == nil {
		reg = monitoring.Default.NewRegistry("libbeat")
	}

	var out outputs.Group
	if !(*publishDisabled) {
		var err error

		if !outcfg.IsSet() {
			msg := "No outputs are defined. Please define one under the output section."
			logp.Info(msg)
			return nil, errors.New(msg)
		}

		// TODO: add support to unload/reassign outStats on output reloading
		outReg := reg.NewRegistry("output")
		outStats := outputs.MakeStats(outReg)

		out, err = outputs.Load(beatInfo, &outStats, outcfg.Name(), outcfg.Config())
		if err != nil {
			return nil, err
		}

		monitoring.NewString(outReg, "type").Set(outcfg.Name())
	}

	name := shipper.Name
	if name == "" {
		name = beatInfo.Hostname
	}

	settings := pipeline.Settings{
		WaitClose:     0,
		WaitCloseMode: pipeline.NoWaitOnClose,
		Disabled:      *publishDisabled,
		Processors:    processors,
		Annotations: pipeline.Annotations{
			Event: shipper.EventMetadata,
			Beat: common.MapStr{
				"name":     name,
				"hostname": beatInfo.Hostname,
				"version":  beatInfo.Version,
			},
		},
	}

	brokerType := "mem"
	if b := shipper.Queue.Name(); b != "" {
		brokerType = b
	}

	brokerFactory := broker.FindFactory(brokerType)
	if brokerFactory == nil {
		return nil, fmt.Errorf("'%v' is no valid queue type", brokerType)
	}

	brokerConfig := shipper.Queue.Config()
	if brokerConfig == nil {
		brokerConfig = common.NewConfig()
	}

	p, err := pipeline.New(
		monitoring.Default.GetRegistry("libbeat"),
		func(eventer broker.Eventer) (broker.Broker, error) {
			return brokerFactory(eventer, brokerConfig)
		},
		out, settings,
	)
	if err != nil {
		return nil, err
	}

	logp.Info("Publisher name: %s", name)
	return p, err
}
