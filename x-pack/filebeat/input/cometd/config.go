// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import "fmt"

type config struct {
	ChannelName string      `config:"channel_name" validate:"required"`
	Auth        *authConfig `config:"auth"`
}

func (c *config) Validate() error {
	if c.ChannelName == "" {
		return fmt.Errorf("no channel name was configured or detected")
	}
	return nil
}

func defaultConfig() config {
	var c config
	c.ChannelName = "cometd-channel"
	return c
}
