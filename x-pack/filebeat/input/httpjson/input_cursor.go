// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/config"
)

type cursorInput struct {
	*input
}

func cursorConfigure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	conf := config.Default()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, nil, err
	}
	return newCursorInput(conf)
}

func newCursorInput(config config.Config) ([]cursor.Source, cursor.Input, error) {
	input, err := newInput(config)
	if err != nil {
		return nil, nil, err
	}
	return nil, &cursorInput{input: input}, nil
}

func (in *cursorInput) Test(cursor.Source, v2.TestContext) error {
	return in.test()
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *cursorInput) Run(
	ctx v2.Context,
	_ cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	return in.run(ctx, publisher, &cursor)
}
