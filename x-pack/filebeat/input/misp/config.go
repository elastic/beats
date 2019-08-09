// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package misp

import "github.com/pkg/errors"

type config struct {
	ServerName string `config:"server_name" validate:"required"`
	Url        string `config:"url" validate:"required"`
}

func (c *config) Validate() error {
	if c.ServerName == "" || c.Url == "" {
		return errors.New("Both server_name and url are required for misp input")
	}
	return nil
}

func defaultConfig() config {
	var c config
	return c
}
