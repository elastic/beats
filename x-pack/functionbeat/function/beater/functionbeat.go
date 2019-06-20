// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/x-pack/functionbeat/function/config"
	"github.com/elastic/beats/x-pack/functionbeat/function/core"
	_ "github.com/elastic/beats/x-pack/functionbeat/function/include"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/libbeat/licenser"
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
	ctx      context.Context
	log      *logp.Logger
	cancel   context.CancelFunc
	Provider provider.Provider
	Config   *config.Config
}

// New creates an instance of functionbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := &config.DefaultConfig
	if err := cfg.Unpack(c); err != nil {
		return nil, fmt.Errorf("error reading config file: %+v", err)
	}

	provider, err := provider.NewProvider(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	bt := &Functionbeat{
		ctx:      ctx,
		cancel:   cancel,
		log:      logp.NewLogger("functionbeat"),
		Provider: provider,
		Config:   c,
	}
	return bt, nil
}

// Run starts functionbeat.
func (bt *Functionbeat) Run(b *beat.Beat) error {
	defer bt.cancel()
	bt.log.Info("Functionbeat is running")
	defer bt.log.Info("Functionbeat stopped running")

	manager, err := licenser.Create(&b.Config.Output, refreshDelay, graceDelay)
	if err != nil {
		return errors.Wrap(err, "could not create the license manager")
	}
	manager.Start()
	defer manager.Stop()

	// Wait until we receive the initial license.
	if err := licenser.WaitForLicense(bt.ctx, bt.log, manager, licenser.BasicAndAboveOrTrial); err != nil {
		return err
	}

	clientFactory := makeClientFactory(bt.log, manager, b.Publisher)

	enabledFunctions := bt.enabledFunctions()
	bt.log.Infof("Functionbeat is configuring enabled functions: %s", strings.Join(enabledFunctions, ", "))
	// Create a client per function and wrap them into a runnable function by the coordinator.
	functions, err := bt.Provider.CreateFunctions(clientFactory, enabledFunctions)
	if err != nil {
		return fmt.Errorf("error when creating the functions, error: %+v", err)
	}

	// manages the goroutine related to the function handlers, if an error occurs and its not handled
	// by the function itself, it will reach the coordinator, we log the error and shutdown beats.
	// When an error reach the coordinator we assume that we cannot recover from it and we initiate
	// a shutdown and return an aggregated errors.
	coordinator := core.NewCoordinator(logp.NewLogger("coordinator"), functions...)
	err = coordinator.Run(bt.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (bt *Functionbeat) enabledFunctions() (values []string) {
	raw, found := os.LookupEnv("ENABLED_FUNCTIONS")
	if !found {
		return values
	}
	return strings.Split(raw, ",")
}

// Stop stops functionbeat.
func (bt *Functionbeat) Stop() {
	bt.log.Info("Functionbeat is stopping")
	defer bt.log.Info("Functionbeat is stopped")
	bt.cancel()
}

func makeClientFactory(log *logp.Logger, manager *licenser.Manager, pipeline beat.Pipeline) func(*common.Config) (core.Client, error) {
	// Each function has his own client to the publisher pipeline,
	// publish operation will block the calling thread, when the method unwrap we have received the
	// ACK for the batch.
	return func(cfg *common.Config) (core.Client, error) {
		c := struct {
			Processors           processors.PluginConfig `config:"processors"`
			common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
		}{}

		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}

		processors, err := processors.New(c.Processors)
		if err != nil {
			return nil, err
		}

		client, err := core.NewSyncClient(log, pipeline, beat.ClientConfig{
			PublishMode: beat.GuaranteedSend,
			Processing: beat.ProcessingConfig{
				Processor:     processors,
				EventMetadata: c.EventMetadata,
			},
		})

		if err != nil {
			return nil, err
		}

		// Make the client aware of the current license, the client will accept sending events to the
		// pipeline until the client is closed or if the license change and is not valid.
		licenseAware := core.NewLicenseAwareClient(client, licenser.BasicAndAboveOrTrial)
		if err := manager.AddWatcher(licenseAware); err != nil {
			return nil, err
		}

		return licenseAware, nil
	}
}
