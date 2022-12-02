// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"errors"
	"fmt"
	"strings"
)

// mode represents the processing mode (original, ecs, ecs_and_original).
type mode uint8

const (
	originalMode       mode = iota // originalMode generates the fields specified in the format string.
	ecsMode                        // ecsMode maps the original fields to ECS and removes the original field if it was mapped.
	ecsAndOriginalMode             // ecsAndOriginalMode maps the original fields to ECS and retains all the original fields.
)

var modeStrings = map[mode]string{
	originalMode:       "original",
	ecsMode:            "ecs",
	ecsAndOriginalMode: "ecs_and_original",
}

func (m *mode) Unpack(s string) error {
	for modeConst, modeStr := range modeStrings {
		if strings.EqualFold(modeStr, s) {
			*m = modeConst
			return nil
		}
	}
	return fmt.Errorf("invalid mode type %q for "+procName, s)
}

func (m *mode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	return m.Unpack(str)
}

func (m *mode) String() string {
	if m == nil {
		return "<nil>"
	}
	if s, found := modeStrings[*m]; found {
		return s
	}
	return "unknown mode"
}

// config contains the configuration options for the processor.
type config struct {
	Format        formats `config:"format" validate:"required"` // VPC flow log format. In config, it can accept a string or list of strings. Each format must have a unique number of fields to enable matching it to a flow log message.
	Mode          mode    `config:"mode"`                       // Mode controls what fields are generated.
	Field         string  `config:"field"`                      // Source field containing the VPC flow log message.
	TargetField   string  `config:"target_field"`               // Target field for the VPC flow log object. This applies only to the original VPC flow log fields. ECS fields are written to the standard location.
	IgnoreMissing bool    `config:"ignore_missing"`             // Ignore missing source field.
	IgnoreFailure bool    `config:"ignore_failure"`             // Ignore failures while parsing and transforming the flow log message.
	ID            string  `config:"id"`                         // Instance ID for debugging purposes.
}

// Validate validates the format strings. Each format must have a unique number
// of fields.
func (c *config) Validate() error {
	counts := map[int]struct{}{}
	for _, format := range c.Format {
		fields, err := parseFormat(format)
		if err != nil {
			return err
		}

		_, found := counts[len(fields)]
		if found {
			return fmt.Errorf("each format must have a unique number of fields")
		}
		counts[len(fields)] = struct{}{}
	}
	return nil
}

func defaultConfig() config {
	return config{
		Mode:        ecsMode,
		Field:       "message",
		TargetField: "aws.vpcflow",
	}
}

// parseFormat parses VPC flow log format string and returns an ordered list of
// the expected fields.
func parseFormat(format string) ([]vpcFlowField, error) {
	tokens := strings.Fields(format)
	if len(tokens) == 0 {
		return nil, errors.New("format must contain at lease one field")
	}

	fields := make([]vpcFlowField, 0, len(tokens))
	for _, token := range tokens {
		// Elastic uses underscores in field names rather than dashes.
		underscoreToken := strings.ReplaceAll(token, "-", "_")

		field, found := nameToFieldMap[underscoreToken]
		if !found {
			return nil, fmt.Errorf("unknown field %q", token)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

type formats []string

func (f *formats) Unpack(value interface{}) error {
	switch v := value.(type) {
	case string:
		*f = []string{v}
	case []string:
		*f = v
	case []interface{}:
		list := make([]string, 0, len(v))
		for _, ifc := range v {
			s, ok := ifc.(string)
			if !ok {
				return fmt.Errorf("format values must be strings, got %T", ifc)
			}
			list = append(list, s)
		}
		*f = list
	default:
		return fmt.Errorf("format must be a string or list of strings, got %T", v)
	}
	return nil
}
