// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/config"
)

var (
	graceDelay   = 45 * time.Minute
	refreshDelay = 15 * time.Minute
)

// Functionbeat is a beat designed to run under a serverless environment and listen to external triggers,
// each invocation will generate one or more events to Elasticsearch.
//
// Each serverless implementation is different but functionbeat follows a few execution rules.
// - Publishing events from the source to the output is done synchronously.
// - Execution can be suspended.
// - Run on a read only filesystem
// - More execution constraints based on speed and memory usage.
type Functionbeat struct {
	log    *logp.Logger
	Config *config.Config
}

// New creates an instance of functionbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := &config.DefaultConfig
	if err := cfg.Unpack(c); err != nil {
		return nil, fmt.Errorf("error reading config file: %+v", err)
	}

	bt := &Functionbeat{
		log:    logp.NewLogger("functionbeat"),
		Config: c,
	}
	return bt, nil
}

// Run starts functionbeat.
func (bt *Functionbeat) Run(b *beat.Beat) error {
	bt.log.Info("Functionbeat is running")
	defer bt.log.Info("Functionbeat stopped running")

	return nil
}

// Stop stops Functionbeat.
func (bt *Functionbeat) Stop() {}
