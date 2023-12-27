// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TemplateBuilderFactory factory method to call to create a new template builder.
type TemplateBuilderFactory func(*logp.Logger, *conf.C, Provider) (TemplateBuilder, error)

// TemplateBuilder generates templates for a given provider.
type TemplateBuilder interface {
	// RawTemplate returns a deployable template string.
	RawTemplate(string) (string, error)
}
