package publisher

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	"github.com/elastic/beats/libbeat/publisher/queue"
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

	name := beatInfo.Name
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

	queueType := "mem"
	if b := shipper.Queue.Name(); b != "" {
		queueType = b
	}

	queueFactory := queue.FindFactory(queueType)
	if queueFactory == nil {
		return nil, fmt.Errorf("'%v' is no valid queue type", queueType)
	}

	queueConfig := shipper.Queue.Config()
	if queueConfig == nil {
		queueConfig = common.NewConfig()
	}

	p, err := pipeline.New(
		monitoring.Default.GetRegistry("libbeat"),
		func(eventer queue.Eventer) (queue.Queue, error) {
			return queueFactory(eventer, queueConfig)
		},
		out, settings,
	)
	if err != nil {
		return nil, err
	}

	logp.Info("Publisher name: %s", name)
	return p, err
}
