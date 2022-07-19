// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"fmt"
	"time"
)

type config struct {
	AccountName string      `config:"account_name"`
	Auth        authConfig  `config:"auth" validate:"required"`
	Containers  []container `config:"containers" validate:"required"`
}

type container struct {
	Name         string        `config:"name" validate:"required"`
	MaxWorkers   int           `config:"max_workers"`
	Poll         bool          `config:"poll"`
	PollInterval time.Duration `config:"poll_interval"`
}

type authConfig struct {
	SharedCredentials *sharedKeyConfig        `config:"shared_credentials,omitempty"`
	ConnectionString  *connectionStringConfig `config:"connection_string,omitempty"`
}

type connectionStringConfig struct {
	URI string `config:"uri,omitempty"`
}
type sharedKeyConfig struct {
	AccountKey string `config:"account_key"`
}

func (c config) Validate() error {
	for _, v := range c.Containers {
		if v.MaxWorkers > 10000 {
			return fmt.Errorf("batch size should be less than 10000")
		}
	}
	return nil
}

func defaultConfig() config {
	return config{
		AccountName: "some_account",
		Auth: authConfig{
			SharedCredentials: &sharedKeyConfig{
				AccountKey: "some_key",
			},
		},
		Containers: []container{
			{Name: "container1", MaxWorkers: 1, Poll: true, PollInterval: time.Duration(time.Second * 5)},
			{Name: "container2", MaxWorkers: 3, Poll: true, PollInterval: time.Duration(time.Second * 5)},
		},
	}
}
