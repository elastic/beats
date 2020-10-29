// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

type responseConfig struct {
	Transforms transformsConfig `config:"transforms"`
	Pagination transformsConfig `config:"pagination"`
}

func (c *responseConfig) Validate() error {
	if _, err := newResponseTransformsFromConfig(c.Transforms); err != nil {
		return err
	}
	if _, err := newPaginationTransformsFromConfig(c.Transforms); err != nil {
		return err
	}

	return nil
}
