package publisher

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher/broker/membroker"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

const defaultBrokerSize = 8 * 1024

func createPipeline(
	beatInfo common.BeatInfo,
	shipper ShipperConfig,
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

	broker := membroker.NewBroker(queueSize, false)
	settings := pipeline.Settings{}
	p, err := pipeline.New(broker, nil, out, settings)
	if err != nil {
		broker.Close()
	}
	return p, err
}
