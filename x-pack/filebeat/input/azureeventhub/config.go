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

// azureInputConfig is the configuration for the azureeventhub input.
type azureInputConfig struct {
	// EventHubName is the name of the event hub to connect to.
	EventHubName string `config:"eventhub" validate:"required"`
	// ConnectionString is the connection string to connect to the event hub.
	// This is required when using Shared Access Key authentication.
	ConnectionString string `config:"connection_string"`
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

	// ---------------------------------------
	// Authentication configuration
	// ---------------------------------------

	// AuthType specifies the authentication method to use for both Event Hub and Storage Account.
	// If not specified, defaults to connection_string for backwards compatibility.
	// Valid values: connection_string, client_secret, managed_identity
	AuthType string `config:"auth_type"`

	// EventHubNamespace is the fully qualified namespace for the Event Hub.
	// Required when using credential-based authentication (client_secret, managed_identity).
	EventHubNamespace string `config:"eventhub_namespace"`
	// TenantID is the Azure Active Directory tenant ID.
	// Required when using client_secret authentication.
	TenantID string `config:"tenant_id"`
	// ClientID is the Azure Active Directory application (client) ID.
	// Required when using client_secret authentication.
	ClientID string `config:"client_id"`
	// ClientSecret is the Azure Active Directory application client secret.
	// Required when using client_secret authentication.
	ClientSecret string `config:"client_secret"`
	// AuthorityHost is the Azure Active Directory authority host.
	// Optional, defaults to Azure Public Cloud (https://login.microsoftonline.com).
	AuthorityHost string `config:"authority_host"`

	// ManagedIdentityClientID is the client ID for user-assigned managed identity.
	// Optional. If not set, system-assigned managed identity is used.
	// Only used when auth_type is managed_identity.
	ManagedIdentityClientID string `config:"managed_identity_client_id"`

	// LegacySanitizeOptions is a list of sanitization options to apply to messages.
	//
	// The supported options are:
	//
	// * NEW_LINES: replaces new lines with spaces
	// * SINGLE_QUOTES: replaces single quotes with double quotes
	//
	// IMPORTANT: Users should use the `sanitizers` configuration option
	// instead.
	//
	// Instead of using the `sanitize_options` configuration option:
	//
	//     sanitize_options:
	//       - NEW_LINES
	//       - SINGLE_QUOTES
	//
	// use the `sanitizers` configuration option:
	//
	//     sanitizers:
	//       - type: new_lines
	//       - type: single_quotes
	//
	// The `sanitize_options` option is deprecated and will be
	// removed in 9.0 release.
	//
	// Default is an empty list (no sanitization).
	LegacySanitizeOptions []string `config:"sanitize_options"`
	// Sanitizers is a list of sanitizers to apply to messages that
	// contain invalid JSON.
	Sanitizers []SanitizerSpec `config:"sanitizers"`

	// ---------------------------------------
	// input v2 specific configuration options
	// ---------------------------------------

	// MigrateCheckpoint controls if the input should perform the checkpoint information
	// migration from v1 to v2 (processor v2 only). Default is true.
	MigrateCheckpoint bool `config:"migrate_checkpoint"`
	// ProcessorVersion controls the processor version to use.
	// Possible values are v1 and v2 (processor v2 only). The default is v2.
	ProcessorVersion string `config:"processor_version" default:"v2"`
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
	// Transport controls the transport type to use for the event hub connection.
	// Possible values are "amqp" (default) and "websocket".
	// Use "websocket" when connecting through HTTP proxies or when port 5671 is blocked.
	Transport string `config:"transport"`
}

func defaultConfig() azureInputConfig {
	return azureInputConfig{
		// For this release, we use
		// the processor v2 as the default.
		ProcessorVersion: processorV2,
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
		LegacySanitizeOptions: []string{},
		// Default is true to avoid reprocessing data from the start of the retention
		// when v2 replaces v1.
		MigrateCheckpoint: true,
		// Default transport is AMQP for backward compatibility.
		Transport: transportAmqp,
	}
}

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	logger := logp.NewLogger("azureeventhub.config")

	// Normalize authentication method (default to connection_string if empty)
	if conf.AuthType == "" {
		conf.AuthType = AuthTypeConnectionString
	}

	// Validate event hub transport option
	if err := conf.validateEventHubTransport(); err != nil {
		return err
	}

	// Validate the processor version first to ensure it's valid
	if err := conf.validateProcessorVersion(); err != nil {
		return err
	}

	// Validate authentication configuration
	if err := conf.validateAuth(); err != nil {
		return err
	}

	// Validate required fields
	if err := conf.validateRequiredFields(); err != nil {
		return err
	}

	// Normalize and validate storage container
	if err := conf.normalizeAndValidateStorageContainer(logger); err != nil {
		return err
	}

	// Validate processor-specific settings
	if err := conf.validateProcessorSettings(); err != nil {
		return err
	}

	// Validate storage account configuration based on processor version
	if err := conf.validateStorageAccountConfig(logger); err != nil {
		return err
	}

	return nil
}

// validateEventHubTransport validates the event hub transport option.
func (conf *azureInputConfig) validateEventHubTransport() error {
	// Validate transport option
	if conf.Transport != transportAmqp && conf.Transport != transportWebsocket {
		return fmt.Errorf(
			"invalid event hub transport: %s (available transports: %s, %s)",
			conf.Transport,
			transportAmqp,
			transportWebsocket,
		)
	}
	return nil
}

// validateProcessorVersion validates that the processor version is valid.
func (conf *azureInputConfig) validateProcessorVersion() error {
	if conf.ProcessorVersion != processorV1 && conf.ProcessorVersion != processorV2 {
		return fmt.Errorf(
			"invalid processor_version: %s (available versions: %s, %s)",
			conf.ProcessorVersion,
			processorV1,
			processorV2,
		)
	}
	return nil
}

// validateAuth validates the authentication configuration based on auth type.
func (conf *azureInputConfig) validateAuth() error {
	switch conf.AuthType {
	case AuthTypeConnectionString:
		return conf.validateConnectionStringAuth()
	case AuthTypeClientSecret:
		return conf.validateClientSecretAuth()
	case AuthTypeManagedIdentity:
		return conf.validateManagedIdentityAuth()
	default:
		return fmt.Errorf("unknown auth_type: %s (valid values: connection_string, client_secret, managed_identity)", conf.AuthType)
	}
}

// validateConnectionStringAuth validates connection string authentication configuration.
func (conf *azureInputConfig) validateConnectionStringAuth() error {
	// Validate Event Hub connection string configuration
	if conf.ConnectionString == "" {
		return errors.New("connection_string is required when auth_type is empty or set to connection_string")
	}
	connectionStringProperties, err := parseConnectionString(conf.ConnectionString)
	if err != nil {
		return fmt.Errorf("invalid connection string: %w", err)
	}

	// If the connection string contains an entity path, we need to double-check that it matches the event hub name.
	if connectionStringProperties.EntityPath != nil && *connectionStringProperties.EntityPath != conf.EventHubName {
		return fmt.Errorf(
			"invalid config: the entity path (%s) in the connection string does not match event hub name (%s)",
			*connectionStringProperties.EntityPath,
			conf.EventHubName,
		)
	}

	// Validate Storage Account authentication for connection_string auth type
	return conf.validateStorageAccountAuthForConnectionString()
}

// validateStorageAccountAuthForConnectionString validates storage account authentication for connection_string auth type.
func (conf *azureInputConfig) validateStorageAccountAuthForConnectionString() error {
	switch conf.ProcessorVersion {
	case processorV1:
		// Processor v1 requires storage account key
		if conf.SAKey == "" {
			return errors.New("storage_account_key is required when using connection_string authentication with processor v1")
		}
	case processorV2:
		// Processor v2 requires storage account connection string, but it can be auto-constructed
		// from SAName and SAKey later in validation. We don't validate it here.
	}
	return nil
}

// validateClientSecretAuth validates client secret authentication configuration.
func (conf *azureInputConfig) validateClientSecretAuth() error {
	// Validate Event Hub client secret configuration
	if conf.EventHubNamespace == "" {
		return errors.New("eventhub_namespace is required when using client_secret authentication")
	}
	if conf.TenantID == "" {
		return errors.New("tenant_id is required when using client_secret authentication")
	}
	if conf.ClientID == "" {
		return errors.New("client_id is required when using client_secret authentication")
	}
	if conf.ClientSecret == "" {
		return errors.New("client_secret is required when using client_secret authentication")
	}

	// Validate Storage Account authentication for client_secret auth type
	return conf.validateStorageAccountAuthForClientSecret()
}

// validateStorageAccountAuthForClientSecret validates storage account authentication for client_secret auth type.
func (conf *azureInputConfig) validateStorageAccountAuthForClientSecret() error {
	switch conf.ProcessorVersion {
	case processorV1:
		// Processor v1 requires storage account key
		if conf.SAKey == "" {
			return errors.New("storage_account_key is required when using client_secret authentication with processor v1")
		}
	case processorV2:
		// Processor v2 with client_secret auth type: Storage Account uses the same client_secret credentials as Event Hub
		// The client_secret credentials are already validated above for Event Hub
		// The storage account will use the same TenantID, ClientID, and ClientSecret as Event Hub
	}
	return nil
}

// validateManagedIdentityAuth validates managed identity authentication configuration.
func (conf *azureInputConfig) validateManagedIdentityAuth() error {
	// Validate Event Hub namespace is provided (required for credential-based auth)
	if conf.EventHubNamespace == "" {
		return errors.New("eventhub_namespace is required when using managed_identity authentication")
	}

	// ManagedIdentityClientID is optional:
	// - If set, uses user-assigned managed identity
	// - If empty, uses system-assigned managed identity

	// For managed_identity, storage account uses the same credential as Event Hub
	// No additional validation needed for processor v2
	return nil
}

// validateRequiredFields validates that all required fields are present.
func (conf *azureInputConfig) validateRequiredFields() error {
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	if conf.SAName == "" {
		return errors.New("no storage account configured (config: storage_account)")
	}
	return nil
}

// normalizeAndValidateStorageContainer normalizes and validates the storage container name.
func (conf *azureInputConfig) normalizeAndValidateStorageContainer(logger *logp.Logger) error {
	// Set default storage account container name if not provided
	if conf.SAContainer == "" {
		// side effect: set the default storage account container name
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

		// side effect: replace underscores (_) with hyphens (-) in the storage account container name
		conf.SAContainer = strings.ReplaceAll(conf.SAContainer, "_", "-")
		logger.Warnf("replaced underscores (_) with hyphens (-) in the storage account container name (before: %s, now: %s", originalValue, conf.SAContainer)
	}

	// Validate the container name conforms to Azure naming rules
	return storageContainerValidate(conf.SAContainer)
}

// validateProcessorSettings validates processor-specific configuration settings.
func (conf *azureInputConfig) validateProcessorSettings() error {
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
	return nil
}

// validateStorageAccountConfig validates storage account configuration based on processor version.
func (conf *azureInputConfig) validateStorageAccountConfig(logger *logp.Logger) error {
	switch conf.ProcessorVersion {
	case processorV1:
		if conf.SAKey == "" {
			return errors.New("no storage account key configured (config: storage_account_key)")
		}
	case processorV2:
		return conf.validateStorageAccountConfigV2(logger)
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

// validateStorageAccountConfigV2 validates storage account configuration for processor v2.
func (conf *azureInputConfig) validateStorageAccountConfigV2(logger *logp.Logger) error {
	// For processor v2, storage account authentication depends on auth_type:
	// - connection_string: needs SAConnectionString (can be auto-constructed from SAName+SAKey)
	// - client_secret: uses the same credentials as Event Hub, no connection string needed
	if conf.AuthType == AuthTypeConnectionString {
		if conf.SAConnectionString == "" {
			if conf.SAName != "" && conf.SAKey != "" {
				// To avoid breaking changes, and ease the migration from v1 to v2,
				// we can build the connection string using the following settings:
				//
				// - DefaultEndpointsProtocol=https;
				// - AccountName=<SAName>;
				// - AccountKey=<SAKey>;
				// - EndpointSuffix=<determined from authority_host or defaults to core.windows.net>
				//
				// Note: For processor v2, we use authority_host to determine the endpoint suffix
				// instead of the deprecated OverrideEnvironment/resource_manager_endpoint.
				// Users can also provide the storage_account_connection_string directly
				// with the correct EndpointSuffix for their cloud environment.
				storageEndpointSuffix := getStorageEndpointSuffix(conf.AuthorityHost)
				conf.SAConnectionString = fmt.Sprintf(
					"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=%s",
					conf.SAName,
					conf.SAKey,
					storageEndpointSuffix,
				)
				logger.Warn("storage_account_connection_string is not configured, but storage_account and storage_account_key are configured. " +
					"The connection string has been constructed from the storage account and key. " +
					"Please configure storage_account_connection_string directly as storage_account_key is deprecated in processor v2.")
				conf.SAKey = ""
			} else {
				// No connection string and no key, so we can't proceed.
				return errors.New("no storage account connection string configured (config: storage_account_connection_string)")
			}
		}
	}
	// For client_secret auth with processor v2, storage account uses the same credentials
	// No connection string validation needed
	return nil
}

// GetFullyQualifiedEventHubNamespace returns the fully qualified namespace for the Event Hub
// based on the configured authentication type.
func (conf *azureInputConfig) GetFullyQualifiedEventHubNamespace() (string, error) {
	switch conf.AuthType {
	case AuthTypeConnectionString:
		// When using connection_string auth, parse it to get the namespace
		connectionStringProperties, err := parseConnectionString(conf.ConnectionString)
		if err != nil {
			return "", fmt.Errorf("failed to parse connection string: %w", err)
		}
		return connectionStringProperties.FullyQualifiedNamespace, nil
	case AuthTypeClientSecret, AuthTypeManagedIdentity:
		// When using client_secret auth, use EventHubNamespace directly
		if conf.EventHubNamespace == "" {
			return "", fmt.Errorf("eventhub_namespace is required when using client_secret authentication")
		}
		return conf.EventHubNamespace, nil
	default:
		return "", fmt.Errorf("unknown auth_type: %s", conf.AuthType)
	}
}

// checkUnsupportedParams checks if unsupported/deprecated/discouraged parameters are set and logs a warning
func (conf *azureInputConfig) checkUnsupportedParams(logger *logp.Logger) {
	logger = logger.Named("azureeventhub.config")

	// log a warning for each sanitization option not supported
	for _, opt := range conf.LegacySanitizeOptions {
		logger.Warnw("legacy sanitization `sanitize_options` options are deprecated and will be removed in the 9.0 release; use the `sanitizers` option instead", "option", opt)
		err := sanitizeOptionsValidate(opt)
		if err != nil {
			logger.Warnf("%s: %v", opt, err)
		}
	}
	if conf.ProcessorVersion == processorV2 {
		if conf.SAKey != "" {
			logger.Warnf("storage_account_key is not used in processor v2, please remove it from the configuration (config: storage_account_key)")
		}
	}
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
		if !unicode.IsLower(runes[i]) && !unicode.IsNumber(runes[i]) && runes[i] != '-' {
			return fmt.Errorf("rune (%d) of storage_account_container (%s) is not a lowercase letter, number or dash", i, name)
		}
		if runes[i] == '-' && previousRune == runes[i] {
			return fmt.Errorf("consecutive dashes ('-') are not permitted in storage_account_container (%s)", name)
		}
		previousRune = runes[i]
	}

	return nil
}
