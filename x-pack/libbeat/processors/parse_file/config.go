// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parse_file

type config struct {
	Field   string    `config:"field"`
	Exclude *[]string `config:"exclude"`
	Only    *[]string `config:"only"`
}

const defaultField = "file.path"

func (c config) FieldOrDefault() string {
	if c.Field == "" {
		return defaultField
	}
	return c.Field
}

func (c config) GetParsers() []parser {
	// only takes precedence to exclude
	if c.Only != nil {
		return onlyParsers(*c.Only)
	}
	if c.Exclude != nil {
		return filterParsers(*c.Exclude)
	}
	return allParsers
}
