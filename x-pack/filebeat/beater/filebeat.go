// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"fmt"

	fbBeater "github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers"
	xpProviders "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers"
)

func FilebeatAutoDiscoverSetup(reg *autodiscover.Registry) error {
	if err := fbBeater.FilebeatAutoDiscoverSetup(reg); err != nil {
		return fmt.Errorf("error setting up autodiscover: %w", err)
	}
	if err := providers.AddKnownProviders(reg, xpProviders.KnownProviders); err != nil {
		return fmt.Errorf("error setting up x-pack autodiscover providers: %w", err)
	}
	return nil
}
