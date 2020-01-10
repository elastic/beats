// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import (
	"errors"
	"fmt"
)

type azureInputConfig struct {
	ConnectionString string `config:"connection_string" validate:"required"`
	EventHubName     string `config:"eventhub" validate:"required"`
	ConsumerGroup    string `config:"consumer_group"`
	// Azure Storage container to store leases and checkpoints
	SAName      string `config:"storage_account"`
	SAKey       string `config:"storage_account_key"`
	SAContainer string `config:"storage_account_container"`
}

const ephContainerName = "filebeat"

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	if conf.ConnectionString == "" {
		return errors.New("no connection string configured")
	}
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	if conf.SAName == "" || conf.SAKey == "" {
		return errors.New("missing storage account information")
	}
	if conf.SAContainer == "" {
		conf.SAContainer = fmt.Sprintf("%s-%s", ephContainerName, conf.EventHubName)
	}
	return nil
}
