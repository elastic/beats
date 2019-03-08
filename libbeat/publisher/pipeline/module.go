// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pipeline

import (
	"flag"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher/processing"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// Global pipeline module for loading the main pipeline from a configuration object

// command line flags
var publishDisabled = false

const defaultQueueType = "mem"

// Monitors configures visibility for observing state and progress of the
// pipeline.
type Monitors struct {
	Metrics   *monitoring.Registry
	Telemetry *monitoring.Registry
	Logger    *logp.Logger
}

// OutputFactory is used by the publisher pipeline to create an output instance.
// If the group returned can be empty. The pipeline will accept events, but
// eventually block.
type OutputFactory func(outputs.Observer) (string, outputs.Group, error)

func init() {
	flag.BoolVar(&publishDisabled, "N", false, "Disable actual publishing for testing")
}

// Load uses a Config object to create a new complete Pipeline instance with
// configured queue and outputs.
func Load(
	beatInfo beat.Info,
	monitors Monitors,
	config Config,
	processors processing.Supporter,
	makeOutput func(outputs.Observer) (string, outputs.Group, error),
) (*Pipeline, error) {
	log := monitors.Logger
	if log == nil {
		log = logp.L()
	}

	if publishDisabled {
		log.Info("Dry run mode. All output types except the file based one are disabled.")
	}

	name := beatInfo.Name
	settings := Settings{
		WaitClose:     0,
		WaitCloseMode: NoWaitOnClose,
		Processors:    processors,
	}

	queueBuilder, err := createQueueBuilder(config.Queue, monitors)
	if err != nil {
		return nil, err
	}

	out, err := loadOutput(monitors, makeOutput)
	if err != nil {
		return nil, err
	}

	p, err := New(beatInfo, monitors, queueBuilder, out, settings)
	if err != nil {
		return nil, err
	}

	log.Infof("Beat name: %s", name)
	return p, err
}

func loadOutput(
	monitors Monitors,
	makeOutput OutputFactory,
) (outputs.Group, error) {
	log := monitors.Logger
	if log == nil {
		log = logp.L()
	}

	if publishDisabled {
		return outputs.Group{}, nil
	}

	if makeOutput == nil {
		return outputs.Group{}, nil
	}

	var (
		metrics  *monitoring.Registry
		outStats outputs.Observer
	)
	if monitors.Metrics != nil {
		metrics = monitors.Metrics.GetRegistry("output")
		if metrics != nil {
			metrics.Clear()
		} else {
			metrics = monitors.Metrics.NewRegistry("output")
		}
		outStats = outputs.NewStats(metrics)
	}

	outName, out, err := makeOutput(outStats)
	if err != nil {
		return outputs.Fail(err)
	}

	if metrics != nil {
		monitoring.NewString(metrics, "type").Set(outName)
	}
	if monitors.Telemetry != nil {
		telemetry := monitors.Telemetry.GetRegistry("output")
		if telemetry != nil {
			telemetry.Clear()
		} else {
			telemetry = monitors.Telemetry.NewRegistry("output")
		}
		monitoring.NewString(telemetry, "name").Set(outName)
	}

	return out, nil
}

func createQueueBuilder(
	config common.ConfigNamespace,
	monitors Monitors,
) (func(queue.Eventer) (queue.Queue, error), error) {
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

	if monitors.Telemetry != nil {
		queueReg := monitors.Telemetry.NewRegistry("queue")
		monitoring.NewString(queueReg, "name").Set(queueType)
	}

	return func(eventer queue.Eventer) (queue.Queue, error) {
		return queueFactory(eventer, monitors.Logger, queueConfig)
	}, nil
}
