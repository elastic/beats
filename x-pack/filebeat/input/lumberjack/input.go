// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"fmt"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	inputName = "lumberjack"
)

func Plugin() inputv2.Plugin {
	return inputv2.Plugin{
		Name:      inputName,
		Stability: feature.Beta,
		Info:      "Receives data streamed via the Lumberjack protocol.",
		Manager:   inputv2.ConfigureWith(configure),
	}
}

func configure(cfg *conf.C) (inputv2.Input, error) {
	var lumberjackConfig config
	if err := cfg.Unpack(&lumberjackConfig); err != nil {
		return nil, err
	}

	return newLumberjackInput(lumberjackConfig)
}

// lumberjackInput implements the Filebeat input V2 interface. The input is stateless.
type lumberjackInput struct {
	config config
}

var _ inputv2.Input = (*lumberjackInput)(nil)

func newLumberjackInput(lumberjackConfig config) (*lumberjackInput, error) {
	return &lumberjackInput{config: lumberjackConfig}, nil
}

func (i *lumberjackInput) Name() string { return inputName }

func (i *lumberjackInput) Test(inputCtx inputv2.TestContext) error {
	s, err := newServer(i.config, inputCtx.Logger, nil, nil)
	if err != nil {
		return err
	}
	return s.Close()
}

func (i *lumberjackInput) Run(inputCtx inputv2.Context, pipeline beat.Pipeline) error {
	inputCtx.Logger.Info("Starting " + inputName + " input")
	defer inputCtx.Logger.Info(inputName + " input stopped")

	// Create client for publishing events and receive notification of their ACKs.
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: newEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	setGoLumberLogger(inputCtx.Logger.Named("go-lumber"))

	metrics := newInputMetrics(inputCtx.ID, nil)
	defer metrics.Close()

	s, err := newServer(i.config, inputCtx.Logger, client.Publish, metrics)
	if err != nil {
		return err
	}
	defer s.Close()

	// Shutdown the server when cancellation is signaled.
	go func() {
		<-inputCtx.Cancelation.Done()
		s.Close()
	}()

	// Run server until the cancellation signal.
	return s.Run()
}
