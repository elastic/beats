package pipeline

import (
	"errors"
	"flag"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// Global pipeline module for loading the main pipeline from a configuration object

// command line flags
var publishDisabled = false

const defaultQueueType = "mem"

func init() {
	flag.BoolVar(&publishDisabled, "N", false, "Disable actual publishing for testing")
}

// Load uses a Config object to create a new complete Pipeline instance with
// configured queue and outputs.
func Load(
	beatInfo beat.Info,
	reg *monitoring.Registry,
	config Config,
	outcfg common.ConfigNamespace,
) (*Pipeline, error) {
	if publishDisabled {
		logp.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, fmt.Errorf("error initializing processors: %v", err)
	}

	name := beatInfo.Name
	settings := Settings{
		WaitClose:     0,
		WaitCloseMode: NoWaitOnClose,
		Disabled:      publishDisabled,
		Processors:    processors,
		Annotations: Annotations{
			Event: config.EventMetadata,
			Beat: common.MapStr{
				"name":     name,
				"hostname": beatInfo.Hostname,
				"version":  beatInfo.Version,
			},
		},
	}

	queueBuilder, err := createQueueBuilder(config.Queue)
	if err != nil {
		return nil, err
	}

	out, err := loadOutput(beatInfo, reg, outcfg)
	if err != nil {
		return nil, err
	}

	p, err := New(beatInfo, reg, queueBuilder, out, settings)
	if err != nil {
		return nil, err
	}

	logp.Info("Beat name: %s", name)
	return p, err
}

func loadOutput(
	beatInfo beat.Info,
	reg *monitoring.Registry,
	outcfg common.ConfigNamespace,
) (outputs.Group, error) {
	if publishDisabled {
		return outputs.Group{}, nil
	}

	if !outcfg.IsSet() {
		msg := "No outputs are defined. Please define one under the output section."
		logp.Info(msg)
		return outputs.Fail(errors.New(msg))
	}

	// TODO: add support to unload/reassign outStats on output reloading

	var (
		outReg   *monitoring.Registry
		outStats outputs.Observer
	)
	if reg != nil {
		outReg = reg.NewRegistry("output")
		outStats = outputs.NewStats(outReg)
	}

	out, err := outputs.Load(beatInfo, outStats, outcfg.Name(), outcfg.Config())
	if err != nil {
		return outputs.Fail(err)
	}

	if outReg != nil {
		monitoring.NewString(outReg, "type").Set(outcfg.Name())
	}

	return out, nil
}

func createQueueBuilder(config common.ConfigNamespace) (func(queue.Eventer) (queue.Queue, error), error) {
	queueType := defaultQueueType
	if b := config.Name(); b != "" {
		queueType = b
	}

	queueFactory := queue.FindFactory(queueType)
	if queueFactory == nil {
		return nil, fmt.Errorf("'%v' is no valid queue type", queueType)
	}

	queueConfig := config.Config()
	if queueConfig == nil {
		queueConfig = common.NewConfig()
	}

	return func(eventer queue.Eventer) (queue.Queue, error) {
		return queueFactory(eventer, queueConfig)
	}, nil
}
