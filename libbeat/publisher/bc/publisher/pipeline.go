package publisher

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/broker/membroker"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

const defaultBrokerSize = 8 * 1024

func createPipeline(
	beatInfo common.BeatInfo,
	shipper ShipperConfig,
	processors *processors.Processors,
	outcfg common.ConfigNamespace,
) (*pipeline.Pipeline, error) {
	queueSize := defaultBrokerSize
	if qs := shipper.QueueSize; qs != nil {
		if sz := *qs; sz > 0 {
			queueSize = sz
		}
	}

	var out outputs.Group
	if !(*publishDisabled) {
		var err error

		if !outcfg.IsSet() {
			msg := "No outputs are defined. Please define one under the output section."
			logp.Info(msg)
			return nil, errors.New(msg)
		}

		out, err = outputs.Load(beatInfo, outcfg.Name(), outcfg.Config())
		if err != nil {
			return nil, err
		}
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
	broker := membroker.NewBroker(queueSize, false)
	p, err := pipeline.New(broker, out, settings)
	if err != nil {
		broker.Close()
		return nil, err
	}

	logp.Info("Publisher name: %s", name)
	return p, err
}
