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
)

const (
	procName = "parse_aws_vpc_flow_log"
	logName  = "processor." + procName
)

func init() {
	processors.RegisterPlugin(procName, New)
	jsprocessor.RegisterPlugin("ParseAWSVPCFlowLog", New)
}

var (
	errValueNotString = errors.New("field must be a string")
	errInvalidFormat  = errors.New("log did not match the specified format")
)

type processor struct {
	config
	fields          []vpcFlowField
	log             *logp.Logger
	expectedIPCount int
}

// New constructs a new processor built from ucfg config.
func New(cfg *conf.C) (processors.Processor, error) {
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

	fields, err := parseFormat(c.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vpc flow log format: %w", err)
	}

	var ipCount int
	for _, f := range fields {
		if f.Type == ipType {
			ipCount++
		}
	}

	return &processor{config: c, fields: fields, expectedIPCount: ipCount, log: log}, nil
}

func (p *processor) String() string {
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

	itr := newWordIterator(strValue)
	if itr.Count() != len(p.fields) {
		return errInvalidFormat
	}

	var relatedIPs []string
	if p.Mode > originalMode {
		relatedIPs = make([]string, 0, p.expectedIPCount)
	}

	originalFields := make(mapstr.M, len(p.fields))

	for itr.Next() {
		// Read one word.
		value := itr.Word()
		if value == "-" {
			continue
		}
		field := p.fields[itr.WordIndex()]

		// Convert the string the expected type.
		v, err := toType(value, field.Type)
		if err != nil {
			return fmt.Errorf("failed to parse <%v>: %w", field.Name, err)
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

	if _, err = event.PutValue(p.TargetField, originalFields); err != nil {
		return err
	}

	if len(relatedIPs) > 0 {
		if _, err = event.PutValue("related.ip", relatedIPs); err != nil {
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
