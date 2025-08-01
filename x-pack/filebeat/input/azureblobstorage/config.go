// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
<<<<<<< HEAD
=======
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/decoder"
	conf "github.com/elastic/elastic-agent-libs/config"
>>>>>>> a62688919 (Feature: expand Parquet file processing support to GCS and AzureBlobStorage inputs.)
)

// MaxWorkers, Poll, PollInterval, FileSelectors, TimeStampEpoch & ExpandEventListFromField can
// be configured at a global level, which applies to all containers. They can also be configured at individual container levels.
// Container level configurations will always override global level values.
type config struct {
	AccountName              string               `config:"account_name" validate:"required"`
	StorageURL               string               `config:"storage_url"`
	Auth                     authConfig           `config:"auth" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers" validate:"max=5000"`
	Poll                     *bool                `config:"poll"`
	PollInterval             *time.Duration       `config:"poll_interval"`
	Containers               []container          `config:"containers" validate:"required"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig             readerConfig         `config:",inline"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// container contains the config for each specific blob storage container in the root account
type container struct {
	Name                     string               `config:"name" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers" validate:"max=5000"`
	Poll                     *bool                `config:"poll"`
	PollInterval             *time.Duration       `config:"poll_interval"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig             readerConfig         `config:",inline"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// fileSelectorConfig helps filter out azure blobs based on a regex pattern
type fileSelectorConfig struct {
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
}

// readerConfig defines the options for reading the content of an azure container.
type readerConfig struct {
<<<<<<< HEAD
	Parsers  parser.Config `config:",inline"`
	Decoding decoderConfig `config:"decoding"`
=======
	// Parsers contains the configuration for different content parsers (e.g., JSON, XML, CSV).
	Parsers parser.Config `config:",inline"`
	// Decoding specifies options for decoding the content, such as compression.
	Decoding decoder.Config `config:"decoding"`
	// ContentType suggests the MIME type of the blob content, aiding in parsing.
	ContentType string `config:"content_type"`
	// Encoding specifies the character encoding of the blob content (e.g., "UTF-8", "gzip").
	Encoding string `config:"encoding"`
	// OverrideContentType indicates whether to force the ContentType rather than inferring it.
	OverrideContentType bool `config:"override_content_type"`
	// OverrideEncoding indicates whether to force the Encoding rather than inferring it.
	OverrideEncoding bool `config:"override_encoding"`
>>>>>>> a62688919 (Feature: expand Parquet file processing support to GCS and AzureBlobStorage inputs.)
}

type authConfig struct {
	SharedCredentials *sharedKeyConfig        `config:"shared_credentials"`
	ConnectionString  *connectionStringConfig `config:"connection_string"`
	OAuth2            *OAuth2Config           `config:"oauth2"`
}

type connectionStringConfig struct {
	URI string `config:"uri"`
}
type sharedKeyConfig struct {
	AccountKey string `config:"account_key"`
}

type OAuth2Config struct {
	ClientID     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`
	TenantID     string `config:"tenant_id"`
	// clientOptions is used internally for testing purposes only
	clientOptions azcore.ClientOptions
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
