// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/elastic/elastic-agent-libs/logp"
)

const ephContainerName = "filebeat"

type azureInputConfig struct {
	// EventHubName is the name of the event hub to connect to.
	EventHubName string `config:"eventhub" validate:"required"`
	// ConnectionString is the connection string to connect to the event hub.
	ConnectionString string `config:"connection_string" validate:"required"`
	// ConsumerGroup is the name of the consumer group to use.
	ConsumerGroup string `config:"consumer_group"`
	// Azure Storage container to store leases and checkpoints
	SAName string `config:"storage_account" validate:"required"`
	// SAKey is used to connect to the storage account (processor v1 only)
	SAKey string `config:"storage_account_key"`
	// SAConnectionString is used to connect to the storage account (processor v2 only)
	SAConnectionString string `config:"storage_account_connection_string"`
	// SAContainer is the name of the storage account container to store
	// partition ownership and checkpoint information.
	SAContainer string `config:"storage_account_container"`
	// by default the azure public environment is used, to override, users can provide a specific resource manager endpoint
	OverrideEnvironment string `config:"resource_manager_endpoint"`
	// cleanup the log JSON input for known issues, options: SINGLE_QUOTES, NEW_LINES
	SanitizeOptions []string `config:"sanitize_options"`
	// MigrateCheckpoint controls if the input should perform the checkpoint information
	// migration from v1 to v2 (processor v2 only). Default is false.
	MigrateCheckpoint bool `config:"migrate_checkpoint"`
	// ProcessorVersion controls the processor version to use.
	// Possible values are v1 and v2 (processor v2 only). Default is v1.
	ProcessorVersion string `config:"processor_version"`
	// ProcessorUpdateInterval controls how often attempt to claim
	// partitions (processor v2 only). The default value is 10 seconds.
	ProcessorUpdateInterval time.Duration `config:"processor_update_interval"`
	// ProcessorStartPosition Controls the start position for all partitions
	// (processor v2 only). Default is "earliest".
	ProcessorStartPosition string `config:"processor_start_position"`
	// PartitionReceiveTimeout controls the batching of incoming messages together
	// with `PartitionReceiveCount` (processor v2 only). Default is 5s.
	//
	// The partition client waits up to `PartitionReceiveTimeout` or
	// for at least `PartitionReceiveCount` events, then it returns
	// the events it has received.
	PartitionReceiveTimeout time.Duration `config:"partition_receive_timeout"`
	// PartitionReceiveCount controls the batching of incoming messages together
	// with `PartitionReceiveTimeout` (processor v2 only). Default is 100.
	//
	// The partition client waits up to `PartitionReceiveTimeout` or
	// for at least `PartitionReceiveCount` events, then it returns
	// the events it has received.
	PartitionReceiveCount int `config:"partition_receive_count"`
}

func defaultConfig() azureInputConfig {
	return azureInputConfig{
		// For this release, we continue to use
		// the processor v1 as the default.
		ProcessorVersion: processorV1,
		// Controls how often attempt to claim partitions.
		ProcessorUpdateInterval: 10 * time.Second,
		// For backward compatibility with v1,
		// the default start position is "earliest".
		ProcessorStartPosition: startPositionEarliest,
		// Receive timeout and count control how
		// many events we want to receive from
		// the processor before returning.
		PartitionReceiveTimeout: 5 * time.Second,
		PartitionReceiveCount:   100,
		// Default
		SanitizeOptions: []string{},
	}
}

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	logger := logp.NewLogger("azureeventhub.config")
	if conf.ConnectionString == "" {
		return errors.New("no connection string configured")
	}
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	if conf.SAName == "" {
		return errors.New("no storage account configured (config: storage_account)")
	}
	if conf.SAContainer == "" {
		conf.SAContainer = fmt.Sprintf("%s-%s", ephContainerName, conf.EventHubName)
	}
	if strings.Contains(conf.SAContainer, "_") {
		originalValue := conf.SAContainer
		// When a user specifies an event hub name in the input settings,
		// the configuration uses it to compose the storage account (SA) container
		// name (for example, `filebeat-<DATA-STREAM>-<EVENTHUB>`).
		//
		// The event hub allows names with underscores (_) characters, but unfortunately,
		// the SA container does not permit them.
		//
		// So instead of throwing an error to the user, we decided to replace
		// underscores (_) characters with hyphens (-).
		conf.SAContainer = strings.ReplaceAll(conf.SAContainer, "_", "-")
		logger.Warnf("replaced underscores (_) with hyphens (-) in the storage account container name (before: %s, now: %s", originalValue, conf.SAContainer)
	}
	err := storageContainerValidate(conf.SAContainer)
	if err != nil {
		return err
	}

	// log a warning for each sanitization option not supported
	for _, opt := range conf.SanitizeOptions {
		err := sanitizeOptionsValidate(opt)
		if err != nil {
			logger.Warnf("%s: %v", opt, err)
		}
	}

	if conf.ProcessorUpdateInterval < 1*time.Second {
		return errors.New("processor_update_interval must be at least 1 second")
	}
	if conf.PartitionReceiveTimeout < 1*time.Second {
		return errors.New("partition_receive_timeout must be at least 1 second")
	}
	if conf.PartitionReceiveCount < 1 {
		return errors.New("partition_receive_count must be at least 1")
	}
	if conf.ProcessorStartPosition != startPositionEarliest && conf.ProcessorStartPosition != startPositionLatest {
		return fmt.Errorf(
			"invalid processor_start_position: %s (available positions: %s, %s)",
			conf.ProcessorStartPosition,
			startPositionEarliest,
			startPositionLatest,
		)
	}

	switch conf.ProcessorVersion {
	case processorV1:
		if conf.SAKey == "" {
			return errors.New("no storage account key configured (config: storage_account_key)")
		}
	case processorV2:
		if conf.SAKey != "" {
			logger.Warnf("storage_account_key is not used in processor v2, please remove it from the configuration (config: storage_account_key)")
		}
		if conf.SAConnectionString == "" {
			return errors.New("no storage account connection string configured (config: storage_account_connection_string)")
		}
	default:
		return fmt.Errorf(
			"invalid processor_version: %s (available versions: %s, %s)",
			conf.ProcessorVersion,
			processorV1,
			processorV2,
		)
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
