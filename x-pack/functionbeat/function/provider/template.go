// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// TemplateBuilderFactory factory method to call to create a new template builder.
type TemplateBuilderFactory func(*logp.Logger, *common.Config, Provider) (TemplateBuilder, error)

// TemplateBuilder generates templates for a given provider.
type TemplateBuilder interface {
	// RawTemplate returns a deployable template string.
	RawTemplate(string) (string, error)
}
