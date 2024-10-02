// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"encoding/json"
	"errors"
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/x-pack/filebeat/processors/aws_vpcflow/internal/strings"
)

const (
	procName = "parse_aws_vpc_flow_log"
	logName  = "processor." + procName
)

// InitializeModule initializes this module.
func InitializeModule() {
	processors.RegisterPlugin(procName, New)
	jsprocessor.RegisterPlugin("ParseAWSVPCFlowLog", New)
}

var (
	errValueNotString = errors.New("field must be a string")
	errInvalidFormat  = errors.New("log did not match the specified format")
)

type processor struct {
	config
	formats []formatProcessor
}

// New constructs a new processor built from ucfg config.
func New(cfg *conf.C) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %w", err)
	}

	return newParseAWSVPCFlowLog(c)
}

func newParseAWSVPCFlowLog(c config) (*processor, error) {
	log := logp.NewLogger(logName)
	if c.ID != "" {
		log = log.With("instance_id", c.ID)
	}

	// Validate configs that did not pass through go-ucfg.
	if err := c.Validate(); err != nil {
		return nil, err
	}

	var formatProcessors []formatProcessor
	for _, f := range c.Format {
		p, err := newFormatProcessor(c, log, f)
		if err != nil {
			return nil, err
		}
		formatProcessors = append(formatProcessors, *p)
	}

	return &processor{
		config:  c,
		formats: formatProcessors,
	}, nil
}

func (p *processor) String() string {
	// JSON encoding of the config struct should never cause an error.
	json, _ := json.Marshal(p.config)
	return procName + "=" + string(json)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	err := p.run(event)
	if err == nil || p.IgnoreFailure || (p.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound)) {
		return event, nil
	}

	return event, err
}

func (p *processor) run(event *beat.Event) error {
	v, err := event.GetValue(p.Field)
	if err != nil {
		return err
	}

	strValue, ok := v.(string)
	if !ok {
		return errValueNotString
	}

	// Split the string at whitespace without allocating.
	var dst [len(vpcFlowFields)]string
	n, err := strings.Fields(dst[:], strValue)
	if err != nil {
		return errInvalidFormat
	}
	substrings := dst[:n]

	// Find the matching format based on substring count.
	for _, format := range p.formats {
		if len(format.fields) == n {
			return format.process(substrings, event)
		}
	}
	return errInvalidFormat
}

// formatProcessor processes an event using a single VPC flow log format.
type formatProcessor struct {
	config
	log                *logp.Logger
	fields             []vpcFlowField
	originalFieldCount int
	expectedIPCount    int
}

func newFormatProcessor(c config, log *logp.Logger, format string) (*formatProcessor, error) {
	fields, err := parseFormat(format)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vpc flow log format: %w", err)
	}

	originalFieldCount := len(fields)
	if c.Mode == ecsMode {
		for _, f := range fields {
			// If an ECS mapping exists then ECS mode will not include the
			// original field.
			if len(f.ECSMappings) > 0 {
				originalFieldCount--
			}
		}
	}

	var ipCount int
	for _, f := range fields {
		if f.Type == ipType {
			ipCount++
		}
	}

	return &formatProcessor{
		config:             c,
		log:                log,
		fields:             fields,
		originalFieldCount: originalFieldCount,
		expectedIPCount:    ipCount,
	}, nil
}

func (p *formatProcessor) process(substrings []string, event *beat.Event) error {
	originalFields := make(mapstr.M, p.originalFieldCount)

	var relatedIPs []string
	if p.Mode > originalMode {
		// Allocate space for the expected number of IPs assuming all are unique.
		relatedIPs = make([]string, 0, p.expectedIPCount)

		// Preallocate event.type with extra capacity for "allowed" or "denied".
		eventTypes := make([]string, 1, 2)
		eventTypes[0] = "connection"
		if _, err := event.PutValue("event.type", eventTypes); err != nil {
			return err
		}
	}

	// Iterate over the substrings in the source string and apply type
	// conversion and then ECS mappings.
	for i, word := range substrings {
		if word == "-" {
			continue
		}
		field := p.fields[i]

		// Convert the string to the expected type.
		v, err := toType(word, field.Type)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", field.Name, err)
		}

		// Add to the 'original' fields when we are in original mode
		// or ecs_and_original mode. Or if there are no ECS mappings then
		// retain the original field.
		if p.Mode != ecsMode || len(field.ECSMappings) == 0 {
			originalFields[field.Name] = v

			if field.Enrich != nil {
				field.Enrich(originalFields, v)
			}
		}

		// Apply ECS transforms when in ecs or ecs_and_original modes.
		if p.Mode > originalMode {
			for _, mapping := range field.ECSMappings {
				if mapping.Transform == nil {
					if _, err = event.PutValue(mapping.Target, v); err != nil {
						return err
					}
				} else {
					mapping.Transform(mapping.Target, v, event)
				}
			}

			if field.Type == ipType {
				relatedIPs = appendUnique(relatedIPs, v.(string))
			}
		}
	}

	if _, err := event.PutValue(p.TargetField, originalFields); err != nil {
		return err
	}

	if len(relatedIPs) > 0 {
		if _, err := event.PutValue("related.ip", relatedIPs); err != nil {
			return err
		}
	}

	return nil
}

// appendUnique appends a value to the slice if the given value does not already
// exist in the slice. It determines if item is in the slice by iterating over
// all elements in the slice and checking equality.
func appendUnique(s []string, item string) []string {
	for _, existing := range s {
		if item == existing {
			return s
		}
	}
	return append(s, item)
}
