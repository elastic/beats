// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
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
	return fmt.Errorf("invalid mode type <%v> for "+procName, s)
}

func (m *mode) UnmarshalYAML(value *yaml.Node) error {
	return m.Unpack(value.Value)
}

func (m *mode) String() string {
	if s, found := modeStrings[*m]; found {
		return s
	}
	return "unknown mode"
}

// config contains the configuration options for the processor.
type config struct {
	Format        string `config:"format" validate:"required"` // VPC flow log format.
	Mode          mode   `config:"mode"`                       // Mode controls what fields are generated.
	Field         string `config:"field"`                      // Source field containing the VPC flow log message.
	TargetField   string `config:"target_field"`               // Target field for the VPC flow log object. This applies only to the original VPC flow log fields. ECS fields are written to the standard location.
	IgnoreMissing bool   `config:"ignore_missing"`             // Ignore missing source field.
	IgnoreFailure bool   `config:"ignore_failure"`             // Ignore failures while parsing and transforming the flow log message.
	ID            string `config:"id"`                         // Instance ID for debugging purposes.
}

// Validate validates the config settings. It returns an error if the format
// string is invalid.
func (c *config) Validate() error {
	_, err := parseFormat(c.Format)
	return err
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
