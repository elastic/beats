// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import "github.com/menderesk/beats/v7/libbeat/common/cfgtype"

type config struct {
	Field         string            `config:"field"`          // Source field containing the CEF message.
	TargetField   string            `config:"target_field"`   // Target field for the CEF object.
	IgnoreMissing bool              `config:"ignore_missing"` // Ignore missing source field.
	IgnoreFailure bool              `config:"ignore_failure"` // Ignore failures when the source field does not contain a CEF message. Parse errors do not cause failures, but are added to error.message.
	ID            string            `config:"id"`             // Instance ID for debugging purposes.
	ECS           bool              `config:"ecs"`            // Generate ECS fields.
	Timezone      *cfgtype.Timezone `config:"timezone"`       // Timezone used when parsing timestamps that do not contain a time zone or offset.
}

func defaultConfig() config {
	return config{
		Field:       "message",
		TargetField: "cef",
		ECS:         true,
	}
}
