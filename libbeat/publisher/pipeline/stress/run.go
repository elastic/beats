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

package stress

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type config struct {
	Generate generateConfig  `config:"generate"`
	Pipeline pipeline.Config `config:"pipeline"`
	Output   conf.Namespace  `config:"output"`
}

var defaultConfig = config{
	Generate: defaultGenerateConfig,
}

// RunTests executes the pipeline stress tests. The test stops after the test
// duration has passed, or runs infinitely if duration is <= 0.  The
// configuration passed must contain the generator settings, the queue setting
// and the test output settings, used to drive the test. If `errors` is not
// nil, internal errors are reported to this callback. A watchdog checking for
// progress is only started if the `errors` callback is set.
// RunTests returns and error if test setup failed, but without `errors` some
// internal errors might not visible.
func RunTests(
	info beat.Info,
	duration time.Duration,
	cfg *conf.C,
	errors func(err error),
) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return fmt.Errorf("unpacking config failed: %v", err)
	}

	log := logp.L()

	processing, err := processing.MakeDefaultSupport(false)(info, log, cfg)
	if err != nil {
		return err
	}

	pipeline, err := pipeline.Load(info,
		pipeline.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log,
		},
		config.Pipeline,
		processing,
		func(stat outputs.Observer) (string, outputs.Group, error) {
			cfg := config.Output
			out, err := outputs.Load(nil, info, stat, cfg.Name(), cfg.Config())
			return cfg.Name(), out, err
		},
	)
	if err != nil {
		return fmt.Errorf("loading pipeline failed: %+v", err)
	}
	defer func() {
		log.Info("Stop pipeline")
		pipeline.Close()
		log.Info("pipeline closed")
	}()

	cs := newCloseSignaler()

	// waitGroup for active generators
	var genWG sync.WaitGroup
	defer genWG.Wait() // block shutdown until all generators have quit

	for i := 0; i < config.Generate.Worker; i++ {
		i := i
		withWG(&genWG, func() {
			err := generate(cs, pipeline, config.Generate, i, errors)
			if err != nil {
				log.Errorf("Generator failed with: %v", err)
			}
		})
	}

	if duration > 0 {
		// Note: don't care about the go-routine leaking (for now)
		go func() {
			time.Sleep(duration)
			cs.Close()
		}()
	}

	return nil
}

func withWG(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}
