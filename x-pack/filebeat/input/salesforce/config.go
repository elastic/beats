// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"time"
)

type config struct {
	Interval time.Duration `config:"interval" validate:"required"`
	Auth     *authConfig   `config:"auth"`
	Url      string        `config:"url" validate:"required"`
	Version  int           `config:"version" validate:"required"`
	Query    *QueryConfig  `config:"query"`
}

func (c *config) Validate() error {
	switch {
	case c.Url == "":
		return errors.New("no instance url was configured or detected")
	case c.Interval == 0:
		return fmt.Errorf("please provide a valid interval %d", c.Interval)
	case c.Version <= 0:
		return fmt.Errorf("please provide a valid version")
	}

	return nil
}

type QueryConfig struct {
	SOQL    string    `config:"soql"` // NOTE(SS): Only for testing purpose
	Default *valueTpl `config:"default"`
	Value   *valueTpl `config:"value"`
}
