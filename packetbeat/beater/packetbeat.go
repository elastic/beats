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
	"github.com/elastic/beats/v7/packetbeat/protos"

	// Add packetbeat default processors
	_ "github.com/elastic/beats/v7/packetbeat/processor/add_kubernetes_metadata"
)

// this is mainly a limitation to ensure that we never deadlock
// after exiting the main select loop in centrally managed packetbeat
// in order to ensure we don't block on a channel write we make sure
// that the errors channel propagated back from the sniffers has a buffer
// that's equal to the number of sniffers that we can run, that way, if
// exiting and we throw a whole bunch of errors for some reason, each
// sniffer can write out the error even though the main loop has already
// exited with the result of the first error
var maxSniffers = 100

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
	config  *common.Config
	factory *processorFactory
	done    chan struct{}
}

func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	configurator := config.NewAgentConfig
	if !b.Manager.Enabled() {
		configurator = initialConfig().FromStatic
	}

	// Install Npcap if needed. This need to happen before any other
	// work on Windows, including config checking, because that involves
	// probing interfaces.
	err := installNpcap(b)
	if err != nil {
		return nil, err
	}

	factory := newProcessorFactory(b.Info.Name, make(chan error, maxSniffers), b, configurator)
	if err := factory.CheckConfig(rawConfig); err != nil {
		return nil, err
	}

	return &packetbeat{
		config:  rawConfig,
		factory: factory,
		done:    make(chan struct{}),
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
	logp.Debug("main", "Waiting for the runner to finish")

	// Start the manager after all the hooks are registered and terminates when
	// the function return.
	if err := b.Manager.Start(); err != nil {
		return err
	}

	defer func() {
		runner.Stop()
		b.Manager.Stop()
	}()

	for {
		select {
		case <-pb.done:
			return nil
		case err := <-factory.err:
			// when we're managed we don't want
			// to stop if the sniffer(s) exited without an error
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
