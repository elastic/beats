// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"errors"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
)

var (
	// ErrProviderUnknown is an error that indicates the provider type is not known.
	ErrProviderUnknown = errors.New("identity: unknown provider type")
)

type conf struct {
	Provider string `config:"provider" validate:"required"`
}

func (c *conf) Validate() error {
	if !provider.Has(c.Provider) {
		return ErrProviderUnknown
	}

	return nil
}
