// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"errors"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
)

// MaxWorkers, Poll, PollInterval, FileSelectors, TimeStampEpoch & ExpandEventListFromField can
// be configured at a global level, which applies to all containers. They can also be configured at individual container levels.
// Container level configurations will always override global level values.
type config struct {
	// AccountName is the name of the Azure Storage account.
	AccountName string `config:"account_name" validate:"required"`
	// StorageURL is the base URL for the Azure Storage service.
	StorageURL string `config:"storage_url"`
	// Auth contains the authentication configuration for accessing the Azure Storage account.
	Auth authConfig `config:"auth" validate:"required"`
	// MaxWorkers defines the maximum number of concurrent workers for processing blobs.
	// It can be set globally or overridden at the container level.
	MaxWorkers *int `config:"max_workers" validate:"max=5000"`
	// Poll enables or disables polling for new blobs in the storage account.
	// It can be set globally or overridden at the container level.
	Poll *bool `config:"poll"`
	// PollInterval specifies the duration between polling attempts for new blobs.
	// It can be set globally or overridden at the container level.
	PollInterval *time.Duration `config:"poll_interval"`
	// Containers is a list of individual container configurations within the storage account.
	Containers []container `config:"containers" validate:"required"`
	// FileSelectors is a list of rules to filter blobs based on regex patterns.
	// These rules can be set globally or overridden at the container level.
	FileSelectors []fileSelectorConfig `config:"file_selectors"`
	// ReaderConfig defines global options for how blob content is read and parsed.
	ReaderConfig readerConfig `config:",inline"`
	// TimeStampEpoch defines a custom epoch timestamp for events.
	// It can be set globally or overridden at the container level.
	TimeStampEpoch *int64 `config:"timestamp_epoch"`
	// ExpandEventListFromField specifies a field from which to expand event lists.
	// It can be set globally or overridden at the container level.
	ExpandEventListFromField string `config:"expand_event_list_from_field"`
}

// container contains the config for each specific blob storage container in the root account.
type container struct {
	// Name is the name of the individual Azure blob storage container.
	Name string `config:"name" validate:"required"`
	// MaxWorkers defines the maximum number of concurrent workers for processing blobs within this specific container.
	// This value overrides the global MaxWorkers setting.
	MaxWorkers *int `config:"max_workers" validate:"max=5000"`
	// Poll enables or disables polling for new blobs within this specific container.
	// This value overrides the global Poll setting.
	Poll *bool `config:"poll"`
	// PollInterval specifies the duration between polling attempts for new blobs within this specific container.
	// This value overrides the global PollInterval setting.
	PollInterval *time.Duration `config:"poll_interval"`
	// FileSelectors is a list of rules to filter blobs based on regex patterns specific to this container.
	// These rules override any global FileSelectors.
	FileSelectors []fileSelectorConfig `config:"file_selectors"`
	// ReaderConfig defines options for how blob content is read and parsed for this specific container.
	// This configuration overrides global ReaderConfig settings.
	ReaderConfig readerConfig `config:",inline"`
	// TimeStampEpoch defines a custom epoch timestamp for events specific to this container.
	// This value overrides the global TimeStampEpoch setting.
	TimeStampEpoch *int64 `config:"timestamp_epoch"`
	// ExpandEventListFromField specifies a field from which to expand event lists for this specific container.
	// This value overrides the global ExpandEventListFromField setting.
	ExpandEventListFromField string `config:"expand_event_list_from_field"`
}

// fileSelectorConfig helps filter out Azure blobs based on a regex pattern.
type fileSelectorConfig struct {
	// Regex is the regular expression pattern used to match blob names.
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
}

// readerConfig defines the options for reading the content of an Azure container.
type readerConfig struct {
	// Parsers contains the configuration for different content parsers (e.g., JSON, XML, CSV).
	Parsers parser.Config `config:",inline"`
	// Decoding specifies options for decoding the content, such as compression.
	Decoding decoderConfig `config:"decoding"`
	// ContentType suggests the MIME type of the blob content, aiding in parsing.
	ContentType string `config:"content_type"`
	// Encoding specifies the character encoding of the blob content (e.g., "UTF-8", "gzip").
	Encoding string `config:"encoding"`
	// OverrideContentType indicates whether to force the ContentType rather than inferring it.
	OverrideContentType bool `config:"override_content_type"`
	// OverrideEncoding indicates whether to force the Encoding rather than inferring it.
	OverrideEncoding bool `config:"override_encoding"`
}

// authConfig defines the various authentication methods for connecting to Azure Storage.
// Only one authentication method should be configured.
type authConfig struct {
	// SharedCredentials uses an account name and shared key for authentication.
	SharedCredentials *sharedKeyConfig `config:"shared_credentials"`
	// ConnectionString uses a full connection string for authentication.
	ConnectionString *connectionStringConfig `config:"connection_string"`
	// OAuth2 uses OAuth 2.0 for authentication, typically with Azure Active Directory.
	OAuth2 *OAuth2Config `config:"oauth2"`
}

// connectionStringConfig holds the details for connection string-based authentication.
type connectionStringConfig struct {
	// URI is the Azure Storage connection string.
	URI string `config:"uri"`
}

// sharedKeyConfig holds the details for shared key-based authentication.
type sharedKeyConfig struct {
	// AccountKey is the shared access key for the Azure Storage account.
	AccountKey string `config:"account_key"`
}

// OAuth2Config holds the details for OAuth 2.0 authentication.
type OAuth2Config struct {
	// ClientID is the application (client) ID for OAuth 2.0 authentication.
	ClientID string `config:"client_id"`
	// ClientSecret is the application client secret for OAuth 2.0 authentication.
	ClientSecret string `config:"client_secret"`
	// TenantID is the Azure Active Directory tenant ID for OAuth 2.0 authentication.
	TenantID string `config:"tenant_id"`
	// clientOptions is used internally for testing purposes only and should not be configured by users.
	clientOptions azcore.ClientOptions
}

// isConfigEmpty checks if the provided configuration value is empty.
// It uses reflection to determine if the value is empty based on its kind.
func isConfigEmpty[T any](value T) bool {
	return isEmpty(reflect.ValueOf(value))
}

// isEmpty checks if a reflect.Value is empty.
// It handles various types including pointers, slices, maps, structs, arrays, and basic types.
// It returns true if the value is empty, false otherwise.
func isEmpty(v reflect.Value) bool {
	// Handles cases like reflect.ValueOf(nil) where nil is untyped,
	// or an uninitialized interface variable.
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		// v.IsNil() checks if the pointer or interface is nil.
		// If it is nil, we consider it empty and return.
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem())

	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0

	case reflect.Struct:
		// Recursively check each field.
		for i := 0; i < v.NumField(); i++ {
			if !isEmpty(v.Field(i)) {
				return false
			}
		}
		return true

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isEmpty(v.Index(i)) {
				return false
			}
		}
		return true

	// 'default:' handles basic types like int, string, bool, float, complex etc.
	default:
		return v.IsZero()
	}
}

func defaultConfig() config {
	return config{
		AccountName: "some_account",
	}
}

func (c config) Validate() error {
	if c.Auth.OAuth2 != nil && (c.Auth.OAuth2.ClientID == "" || c.Auth.OAuth2.ClientSecret == "" || c.Auth.OAuth2.TenantID == "") {
		return errors.New("client_id, client_secret and tenant_id are required for OAuth2 auth")
	}
	return nil
}
