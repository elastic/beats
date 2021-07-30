// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
)

type cursorInput struct{}

func (cursorInput) Name() string {
	return "httpjson-cursor"
}

type source struct {
	config config
}

func (src source) Name() string {
	return src.config.URL.String()
}

func cursorConfigure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	conf := newDefaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, nil, err
	}

	sources, inp := newCursorInput(conf)
	return sources, inp, nil
}

func newCursorInput(config config) ([]cursor.Source, cursor.Input) {
	// we only allow one url per config, if we wanted to allow more than one
	// each source should hold only one url
	return []cursor.Source{&source{config: config}}, &cursorInput{}
}

func (in *cursorInput) Test(src cursor.Source, _ v2.TestContext) error {
	return test((src.(*source)).config.URL.URL)
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *cursorInput) Run(
	ctx v2.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	s := src.(*source)
	return run(ctx, s.config, publisher, &cursor)
}
