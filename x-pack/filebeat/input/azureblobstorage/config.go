// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/match"
)

// MaxWorkers, Poll, PollInterval & PathPrefix can be configured at a global level,
// which applies to all containers. They can also be configured at individual container levels.
// Container level configurations will always override global level values.
type config struct {
	AccountName              string               `config:"account_name" validate:"required"`
	StorageURL               string               `config:"storage_url,omitempty"`
	Auth                     authConfig           `config:"auth" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers,omitempty" validate:"max=5000"`
	Poll                     *bool                `config:"poll,omitempty"`
	PollInterval             *time.Duration       `config:"poll_interval,omitempty"`
	Containers               []container          `config:"containers" validate:"required"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch,omitempty"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// container contains the config for each specific blob storage container in the root account
type container struct {
	Name                     string               `config:"name" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers,omitempty" validate:"max=5000"`
	Poll                     *bool                `config:"poll,omitempty"`
	PollInterval             *time.Duration       `config:"poll_interval,omitempty"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch,omitempty"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// fileSelectorConfig defines reader configuration that applies to a subset
// of azure blobs whose name matches the given regex.
type fileSelectorConfig struct {
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
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
