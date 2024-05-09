// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package benchmark

import "fmt"

type benchmarkConfig struct {
	Message string `config:"message"`
	Count   uint64 `config:"count"`
	Threads uint8  `config:"threads"`
	Eps     uint64 `config:"eps"`
}

var (
	defaultConfig = benchmarkConfig{
		Message: "generic benchmark message",
		Threads: 1,
	}
)

func (c *benchmarkConfig) Validate() error {
	if c.Count > 0 && c.Eps > 0 {
		return fmt.Errorf("only one of count or eps may be specified, not both")
	}
	if c.Message == "" {
		return fmt.Errorf("message must be specified")
	}
	return nil
}
