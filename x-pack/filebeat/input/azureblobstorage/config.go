// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"time"
)

// MaxWorkers, Poll & PollInterval can be configured at a global level,
// which applies to all containers, as well as at the container level.
// Container level configurations will always override global level values.
type config struct {
	AccountName  string         `config:"account_name" validate:"required"`
	StorageURL   string         `config:"storage_url,omitempty"`
	Auth         authConfig     `config:"auth" validate:"required"`
	MaxWorkers   *int           `config:"max_workers,omitempty" validate:"max=5000"`
	Poll         *bool          `config:"poll,omitempty"`
	PollInterval *time.Duration `config:"poll_interval,omitempty"`
	Containers   []container    `config:"containers" validate:"required"`
}

// container contains the config for each specific blob storage container in the root account
type container struct {
	Name         string         `config:"name" validate:"required"`
	MaxWorkers   *int           `config:"max_workers,omitempty" validate:"max=5000"`
	Poll         *bool          `config:"poll,omitempty"`
	PollInterval *time.Duration `config:"poll_interval,omitempty"`
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

func defaultConfig() config {
	return config{
		AccountName: "some_account",
	}
}
