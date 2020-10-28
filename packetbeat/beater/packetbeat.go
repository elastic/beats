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

package beater

import (
	"flag"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/service"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"

	// Add packetbeat default processors
	_ "github.com/elastic/beats/v7/packetbeat/processor/add_kubernetes_metadata"
)

type flags struct {
	file       *string
	loop       *int
	oneAtAtime *bool
	topSpeed   *bool
	dumpfile   *string
}

var cmdLineArgs flags

func init() {
	cmdLineArgs = flags{
		file:       flag.String("I", "", "Read packet data from specified file"),
		loop:       flag.Int("l", 1, "Loop file. 0 - loop forever"),
		oneAtAtime: flag.Bool("O", false, "Read packets one at a time (press Enter)"),
		topSpeed:   flag.Bool("t", false, "Read packets as fast as possible, without sleeping"),
		dumpfile:   flag.String("dump", "", "Write all captured packets to this libpcap file"),
	}
}

func initialConfig() config.Config {
	return config.Config{
		Interfaces: config.InterfacesConfig{
			File:       *cmdLineArgs.file,
			Loop:       *cmdLineArgs.loop,
			TopSpeed:   *cmdLineArgs.topSpeed,
			OneAtATime: *cmdLineArgs.oneAtAtime,
			Dumpfile:   *cmdLineArgs.dumpfile,
		},
	}
}

// Beater object. Contains all objects needed to run the beat
type packetbeat struct {
	config          *common.Config
	factory         *processorFactory
	publisher       *publish.TransactionPublisher
	shutdownTimeout time.Duration
	done            chan struct{}
}

func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	cfg := initialConfig()
	err := rawConfig.Unpack(&cfg)
	if err != nil {
		logp.Err("fails to read the beat config: %v, %v", err, cfg)
		return nil, err
	}

	watcher := procs.ProcessesWatcher{}
	// Enable the process watcher only if capturing live traffic
	if cfg.Interfaces.File == "" {
		err = watcher.Init(cfg.Procs)
		if err != nil {
			logp.Critical(err.Error())
			return nil, err
		}
	} else {
		logp.Info("Process watcher disabled when file input is used")
	}

	publisher, err := publish.NewTransactionPublisher(
		b.Info.Name,
		b.Publisher,
		cfg.IgnoreOutgoing,
		cfg.Interfaces.File == "",
	)
	if err != nil {
		return nil, err
	}

	configurator := config.NewAgentConfig
	if !b.Manager.Enabled() {
		configurator = cfg.FromStatic
	}

	factory := newProcessorFactory(b.Info.Name, make(chan error, 1), publisher, configurator)
	if err := factory.CheckConfig(rawConfig); err != nil {
		return nil, err
	}

	return &packetbeat{
		config:          rawConfig,
		shutdownTimeout: cfg.ShutdownTimeout,
		factory:         factory,
		publisher:       publisher,
		done:            make(chan struct{}),
	}, nil
}

func (pb *packetbeat) Run(b *beat.Beat) error {
	defer func() {
		if service.ProfileEnabled() {
			logp.Debug("main", "Waiting for streams and transactions to expire...")
			time.Sleep(time.Duration(float64(protos.DefaultTransactionExpiration) * 1.2))
			logp.Debug("main", "Streams and transactions should all be expired now.")
		}
	}()

	defer pb.publisher.Stop()

	timeout := pb.shutdownTimeout
	if timeout > 0 {
		defer time.Sleep(timeout)
	}

	if !b.Manager.Enabled() {
		return pb.runStatic(b, pb.factory)
	}
	return pb.runManaged(b, pb.factory)
}

func (pb *packetbeat) runStatic(b *beat.Beat, factory *processorFactory) error {
	runner, err := factory.Create(b.Publisher, pb.config)
	if err != nil {
		return err
	}
	runner.Start()
	defer runner.Stop()

	logp.Debug("main", "Waiting for the runner to finish")

	select {
	case <-pb.done:
	case err := <-factory.err:
		close(pb.done)
		return err
	}
	return nil
}

func (pb *packetbeat) runManaged(b *beat.Beat, factory *processorFactory) error {
	runner := newReloader(management.DebugK, factory, b.Publisher)
	reload.Register.MustRegisterList("inputs", runner)
	defer runner.Stop()

	logp.Debug("main", "Waiting for the runner to finish")

	for {
		select {
		case <-pb.done:
			return nil
		case err := <-factory.err:
			// when we're managed we don't want
			// to stop if the sniffer exited without an error
			// this would happen during a configuration reload
			if err != nil {
				close(pb.done)
				return err
			}
		}
	}
}

// Called by the Beat stop function
func (pb *packetbeat) Stop() {
	logp.Info("Packetbeat send stop signal")
	close(pb.done)
}
