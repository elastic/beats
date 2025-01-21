// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"errors"
)

type infoConfig struct {
	Count uint `config:"count"`
}

var defaultConfig = infoConfig{
	Count: 1,
}

func (c *infoConfig) Validate() error {
	if c.Count == 0 {
		return errors.New("benchmark module 'count' must be greater than 0")
	}
	return nil
}
