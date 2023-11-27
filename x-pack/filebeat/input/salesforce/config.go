// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"fmt"
	"time"
)

type config struct {
	Interval time.Duration `config:"interval" validate:"required"`
	Auth     *authConfig   `config:"auth"`
	Url      string        `config:"url" validate:"required"`
	Soql     *SoqlConfig   `config:"soql"`
	Query    *QueryConfig  `config:"query"`
}

func (c *config) Validate() error {
	if c.Url == "" {
		return fmt.Errorf("no instance url was configured or detected")
	}
	return nil
}

type QueryConfig struct {
	Default *valueTpl `config:"default"`
	Value   *valueTpl `config:"value"`
}

type SoqlConfig struct {
	Query  string          `config:"query"`
	Object string          `config:"object"`
	Fields []string        `config:"fields"`
	Where  WhereTypeConfig `config:"where"`
	Order  OrderConfig     `config:"order"`
}

type WhereTypeConfig struct {
	Or  []WhereConfig `config:"or"`
	And []WhereConfig `config:"and"`
}

type WhereConfig struct {
	Op     string        `config:"op"`
	Values []interface{} `config:"values"`
	Field  string        `config:"field"`
}

type OrderConfig struct {
	By        string `config:"by"`
	Ascending bool   `config:"asc"`
	NullFirst bool   `config:"null_first"`
}
