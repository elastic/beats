// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/multiline"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
)

var (
	ErrNoSuchParser = errors.New("no such parser")
)

// parser transforms or translates the Content attribute of a Message.
// They are able to aggregate two or more Messages into a single one.
type parser interface {
	io.Closer
	Next() (reader.Message, error)
}

type parserConfig struct {
	maxBytes       cfgtype.ByteSize
	lineTerminator readfile.LineTerminator
}

func newParsers(in reader.Reader, pCfg parserConfig, c []common.ConfigNamespace) (parser, error) {
	p := in

	for _, ns := range c {
		name := ns.Name()
		switch name {
		case "multiline":
			var config multiline.Config
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return nil, fmt.Errorf("error while parsing multiline parser config: %+v", err)
			}
			p, err = multiline.New(p, "\n", int(pCfg.maxBytes), &config)
			if err != nil {
				return nil, fmt.Errorf("error while creating multiline parser: %+v", err)
			}
		default:
			return nil, fmt.Errorf("%s: %s", ErrNoSuchParser, name)
		}
	}

	return p, nil
}

func validateParserConfig(pCfg parserConfig, c []common.ConfigNamespace) error {
	for _, ns := range c {
		name := ns.Name()
		switch name {
		case "multiline":
			var config multiline.Config
			cfg := ns.Config()
			err := cfg.Unpack(&config)
			if err != nil {
				return fmt.Errorf("error while parsing multiline parser config: %+v", err)
			}
		default:
			return fmt.Errorf("%s: %s", ErrNoSuchParser, name)
		}
	}

	return nil
}
