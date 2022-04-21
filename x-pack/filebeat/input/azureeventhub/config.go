// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type azureInputConfig struct {
	ConnectionString string `config:"connection_string" validate:"required"`
	EventHubName     string `config:"eventhub" validate:"required"`
	ConsumerGroup    string `config:"consumer_group"`
	// Azure Storage container to store leases and checkpoints
	SAName      string `config:"storage_account"`
	SAKey       string `config:"storage_account_key"`
	SAContainer string `config:"storage_account_container"`
	// by default the azure public environment is used, to override, users can provide a specific resource manager endpoint
	OverrideEnvironment string `config:"resource_manager_endpoint"`
}

const ephContainerName = "filebeat"

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	logger := logp.NewLogger("azureInputConfig.validate")
	if conf.ConnectionString == "" {
		return errors.New("no connection string configured")
	}
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	if conf.SAName == "" || conf.SAKey == "" {
		return errors.New("no storage account or storage account key configured")
	}
	if conf.SAContainer == "" {
		conf.SAContainer = fmt.Sprintf("%s-%s", ephContainerName, conf.EventHubName)
	}
	if strings.Contains(conf.SAContainer, "_") {
		originalValue := conf.SAContainer
		// When a user specifies an event hub name in the input settings,
		// the configuration uses it to compose the storage account (SA) container
		// name (for example, `filebeat-<LOGTYPE>-<EVENTHUB>`).
		//
		// The event hub allows names with underscores (_) characters, but unfortunately,
		// the SA container does not permit them.
		//
		// So instead of throwing an error to the user we decided try to replace (_)
		// characters with hyphens (-).
		conf.SAContainer = strings.ReplaceAll(conf.SAContainer, "_", "-")
		logger.Warnf("removed underscores (_) from the storage account container name (before: %s, now: %s", originalValue, conf.SAContainer)
	}
	err := storageContainerValidate(conf.SAContainer)
	if err != nil {
		return err
	}

	return nil
}

// storageContainerValidate validated the storage_account_container to make sure it is conforming to all the Azure
// naming rules.
// To learn more, please check the Azure documentation visiting:
// https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-containers--blobs--and-metadata#container-names
func storageContainerValidate(name string) error {
	var previousRune rune
	runes := []rune(name)
	length := len(runes)
	if length < 3 {
		return fmt.Errorf("storage_account_container (%s) must be 3 or more characters", name)
	}
	if length > 63 {
		return fmt.Errorf("storage_account_container (%s) must be less than 63 characters", name)
	}
	if !unicode.IsLower(runes[0]) && !unicode.IsNumber(runes[0]) {
		return fmt.Errorf("storage_account_container (%s) must start with a lowercase letter or number", name)
	}
	if !unicode.IsLower(runes[length-1]) && !unicode.IsNumber(runes[length-1]) {
		return fmt.Errorf("storage_account_container (%s) must end with a lowercase letter or number", name)
	}
	for i := 0; i < length; i++ {
		if !unicode.IsLower(runes[i]) && !unicode.IsNumber(runes[i]) && !(runes[i] == '-') {
			return fmt.Errorf("rune %d of storage_account_container (%s) is not a lowercase letter, number or dash", i, name)
		}
		if runes[i] == '-' && previousRune == runes[i] {
			return fmt.Errorf("consecutive dashes ('-') are not permitted in storage_account_container (%s)", name)
		}
		previousRune = runes[i]
	}
	return nil
}
