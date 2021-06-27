// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source"
)

func DefaultConfig() *Config {
	return &Config{
		Sandbox: false,
	}
}

type Config struct {
	Schedule  string                 `config:"schedule"`
	Params    map[string]interface{} `config:"params"`
	RawConfig *common.Config
	Source    *source.Source `config:"source"`
	// Name is optional for lightweight checks but required for browsers
	Name string `config:"name"`
	// Id is optional for lightweight checks but required for browsers
	Id             string   `config:"id"`
	Sandbox        bool     `config:"sandbox"`
	SyntheticsArgs []string `config:"synthetics_args"`
}

var ErrNameRequired = fmt.Errorf("config 'name' must be specified for this monitor")
var ErrIdRequired = fmt.Errorf("config 'id' must be specified for this monitor")
var ErrSourceRequired = fmt.Errorf("config 'source' must be specified for this monitor, if upgrading from a previous experimental version please see our new config docs")

func (c *Config) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.Id == "" {
		return ErrIdRequired
	}

	if c.Source == nil {
		return ErrSourceRequired
	}

	return nil
}
