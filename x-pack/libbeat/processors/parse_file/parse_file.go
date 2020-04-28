// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parse_file

import (
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
)

const (
	processorName = "parse_file"
	// size for mime detection, office file
	// detection requires ~8kb to detect properly
	headerSize = 8192
)

func init() {
	processors.RegisterPlugin(processorName, New)
}

type parseFile struct {
	field   string
	parsers []parser
	log     *logp.Logger
}

const selector = "parse_file"

// New constructs a new add_cloudfoundry_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	var config config
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	log := logp.NewLogger(selector)

	return &parseFile{
		field:   config.FieldOrDefault(),
		parsers: config.GetParsers(),
		log:     log,
	}, nil
}

func (p *parseFile) Run(event *beat.Event) (*beat.Event, error) {
	valI, err := event.GetValue(p.field)
	if err != nil {
		// doesn't have the required file.path value to add more information
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	fileType, fieldSet, found := p.applyParser(val)
	if found {
		event.Fields.DeepUpdate(common.MapStr{
			"file": common.MapStr{
				fileType: fieldSet,
			},
		})
	}
	return event, nil
}

func (p *parseFile) applyParser(path string) (string, common.MapStr, bool) {
	p.log.Debugf("fingerprinting file at %s", path)
	f, err := os.Open(path)
	if err != nil {
		p.log.Errorf("unable to open file at '%s': %v", path, err)
		return "", common.MapStr{}, false
	}
	defer f.Close()

	header := make([]byte, headerSize)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		p.log.Errorf("unable to read file '%s': %v", path, err)
		return "", common.MapStr{}, false
	}
	// reset header read
	if _, err := f.Seek(0, 0); err != nil {
		p.log.Errorf("unable to reset file '%s' after header read: %v", path, err)
		return "", common.MapStr{}, false
	}
	header = header[:n]

	for _, parser := range p.parsers {
		fileParser := parser.Factory()
		if fileParser.Identify(header) {
			name := parser.Name

			p.log.Debugf("file '%s' fingerprinted as '%s'", path, name)

			parsed, err := fileParser.Parse(f)
			if err != nil {
				p.log.Errorf("unable to parse file '%s': %v", path, err)
				return "", common.MapStr{}, false
			}

			return name, parsed, true
		}
	}

	return "", common.MapStr{}, false
}

func (p *parseFile) String() string {
	return processorName
}
