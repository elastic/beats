// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type statelessInput struct {
	config    config
	tlsConfig *tlscommon.TLSConfig
}

func (statelessInput) Name() string {
	return "httpjson-stateless"
}

func statelessConfigure(cfg *common.Config) (stateless.Input, error) {
	conf := newDefaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}
	return newStatelessInput(conf)
}

func newStatelessInput(config config) (*statelessInput, error) {
	tlsConfig, err := newTLSConfig(config)
	if err != nil {
		return nil, err
	}
	return &statelessInput{config: config, tlsConfig: tlsConfig}, nil
}

func (in *statelessInput) Test(v2.TestContext) error {
	return test(in.config.URL.URL)
}

type statelessPublisher struct {
	wrapped stateless.Publisher
}

func (pub statelessPublisher) Publish(event beat.Event, _ interface{}) error {
	pub.wrapped.Publish(event)
	return nil
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *statelessInput) Run(ctx v2.Context, publisher stateless.Publisher) error {
	pub := statelessPublisher{wrapped: publisher}
	return run(ctx, in.config, in.tlsConfig, pub, nil)
}
