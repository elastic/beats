// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type cursorInput struct{}

func (cursorInput) Name() string {
	return "httpjson-cursor"
}

type source struct {
	config    config
	tlsConfig *tlscommon.TLSConfig
}

func (src source) Name() string {
	return src.config.Request.URL.String()
}

func cursorConfigure(cfg *common.Config) ([]inputcursor.Source, inputcursor.Input, error) {
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, nil, err
	}
	return newCursorInput(conf)
}

func newCursorInput(config config) ([]inputcursor.Source, inputcursor.Input, error) {
	tlsConfig, err := newTLSConfig(config)
	if err != nil {
		return nil, nil, err
	}
	// we only allow one url per config, if we wanted to allow more than one
	// each source should hold only one url
	return []inputcursor.Source{
			&source{config: config,
				tlsConfig: tlsConfig,
			},
		},
		&cursorInput{},
		nil
}

func (in *cursorInput) Test(src inputcursor.Source, _ v2.TestContext) error {
	return test((src.(*source)).config.Request.URL.URL)
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *cursorInput) Run(
	ctx v2.Context,
	src inputcursor.Source,
	cursor inputcursor.Cursor,
	publisher inputcursor.Publisher,
) error {
	s := src.(*source)
	return run(ctx, s.config, s.tlsConfig, publisher, &cursor)
}
