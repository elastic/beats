// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/x-pack/beatless/bus"
	"github.com/elastic/beats/x-pack/beatless/config"
)

// Beatless configuration.
type Beatless struct {
	done   chan struct{}
	config config.Config
	log    *logp.Logger

	// TODO: Add registry reference here.
}

// New creates an instance of beatless.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	bt := &Beatless{
		done:   make(chan struct{}),
		config: c,
		log:    logp.NewLogger("beatless"),
	}
	return bt, nil
}

// Run starts beatless.
func (bt *Beatless) Run(b *beat.Beat) error {
	bt.log.Info("beatless is running")
	defer bt.log.Info("beatless stopped running")

	client, err := b.Publisher.Connect()
	if err != nil {
		return err
	}
	defer client.Close()

	// NOTE: Do not review below, this is the minimal to have a working PR.
	bus := bus.New(client)
	// TODO: noop
	bus.Listen()

	// Stop until we are tell to shutdown.
	// TODO this is where the events catcher starts.
	select {
	case <-bt.done:
		// Stop catching events.
	}
	return nil
}

// Stop stops beatless.
func (bt *Beatless) Stop() {
	bt.log.Info("beatless is stopping")
	defer bt.log.Info("beatless is stopped")
	close(bt.done)
}
