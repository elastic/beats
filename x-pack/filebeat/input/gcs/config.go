// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
)

// MaxWorkers, Poll, PollInterval, BucketTimeOut, ParseJSON, FileSelectors, TimeStampEpoch & ExpandEventListFromField
// can be configured at a global level, which applies to all buckets, as well as at the bucket level.
// Bucket level configurations will always override global level values.
type config struct {
	// ProjectId - Defines the project id of the concerned gcs bucket in Google Cloud.
	ProjectId string `config:"project_id" validate:"required"`
	// Auth - Defines the authentication mechanism to be used for accessing the gcs bucket.
	Auth authConfig `config:"auth"`
	// BatchSize - Defines the maximum number of objects that will be fetched from the bucket in a single request.
	BatchSize int `config:"batch_size"`
	// MaxWorkers - Defines the maximum number of go routines that will be spawned.
	MaxWorkers int `config:"max_workers" validate:"max=5000"`
	// Poll - Defines if polling should be performed on the input bucket source.
	Poll bool `config:"poll"`
	// PollInterval - Defines the maximum amount of time to wait before polling for the next batch of objects from the bucket.
	PollInterval time.Duration `config:"poll_interval"`
	// ParseJSON - Informs the publisher whether to parse & objectify json data or not. By default this is set to
	// false, since it can get expensive dealing with highly nested json data.
	ParseJSON bool `config:"parse_json"`
	// Buckets - Defines a list of buckets that will be polled for objects.
	Buckets []bucket `config:"buckets" validate:"required"`
	// FileSelectors - Defines a list of regex patterns that can be used to filter out objects from the bucket.
	FileSelectors []fileSelectorConfig `config:"file_selectors"`
	// ReaderConfig is the default parser and decoder configuration.
	ReaderConfig readerConfig `config:",inline"`
	// TimeStampEpoch - Defines the epoch time in seconds, which is used to filter out objects that are older than the specified timestamp.
	TimeStampEpoch *int64 `config:"timestamp_epoch"`
	// ExpandEventListFromField - Defines the field name that will be used to expand the event into separate events.
	ExpandEventListFromField string `config:"expand_event_list_from_field"`
	// This field is only used for system test purposes, to override the HTTP endpoint.
	AlternativeHost string `config:"alternative_host"`
	// Retry - Defines the retry configuration for the input.
	Retry retryConfig `config:"retry"`
}

// bucket contains the config for each specific object storage bucket in the root account
type bucket struct {
	Name                     string               `config:"name" validate:"required"`
	BatchSize                *int                 `config:"batch_size"`
	MaxWorkers               *int                 `config:"max_workers" validate:"max=5000"`
	Poll                     *bool                `config:"poll"`
	PollInterval             *time.Duration       `config:"poll_interval"`
	ParseJSON                *bool                `config:"parse_json"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig             readerConfig         `config:",inline"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// fileSelectorConfig helps filter out gcs objects based on a regex pattern
type fileSelectorConfig struct {
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
}

// readerConfig defines the options for reading the content of an GCS object.
type readerConfig struct {
	Parsers  parser.Config `config:",inline"`
	Decoding decoderConfig `config:"decoding"`
}

// authConfig defines the authentication mechanism to be used for accessing the gcs bucket.
// If either is configured the 'omitempty' tag will prevent the other option from being serialized in the config.
type authConfig struct {
	CredentialsJSON *jsonCredentialsConfig `config:"credentials_json,omitempty"`
	CredentialsFile *fileCredentialsConfig `config:"credentials_file,omitempty"`
}

type fileCredentialsConfig struct {
	Path string `config:"path"`
}
type jsonCredentialsConfig struct {
	AccountKey string `config:"account_key"`
}

type retryConfig struct {
	// MaxAttempts configures the maximum number of times an API call can be made in the case of retryable errors.
	// For example, if you set MaxAttempts(5), the operation will be attempted up to 5 times total (initial call plus 4 retries).
	// If you set MaxAttempts(1), the operation will be attempted only once and there will be no retries. This setting defaults to 3.
	MaxAttempts int `config:"max_attempts" validate:"min=1"`
	// InitialBackOffDuration is the initial value of the retry period, defaults to 1 second.
	InitialBackOffDuration time.Duration `config:"initial_backoff_duration" validate:"min=1"`
	// MaxBackOffDuration is the maximum value of the retry period, defaults to 30 seconds.
	MaxBackOffDuration time.Duration `config:"max_backoff_duration" validate:"min=2"`
	// BackOffMultiplier is the factor by which the retry period increases. It should be greater than 1 and defaults to 2.
	BackOffMultiplier float64 `config:"backoff_multiplier" validate:"min=1.1"`
}

func (c authConfig) Validate() error {
	// credentials_file
	if c.CredentialsFile != nil {
		_, err := os.Stat(c.CredentialsFile.Path)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("credentials_file is configured, but the file %q cannot be found", c.CredentialsFile.Path)
		} else {
			return nil
		}
	}

	// credentials_json
	if c.CredentialsJSON != nil && len(c.CredentialsJSON.AccountKey) > 0 {
		return nil
	}

	// Application Default Credentials (ADC)
	_, err := google.FindDefaultCredentials(context.Background(), storage.ScopeReadOnly)
	if err == nil {
		return nil
	}

	return fmt.Errorf("no authentication credentials were configured or detected " +
		"(credentials_file, credentials_json, and application default credentials (ADC))")
}

// defaultConfig returns the default configuration for the input
func defaultConfig() config {
	return config{
		MaxWorkers:   1,
		Poll:         true,
		PollInterval: 5 * time.Minute,
		ParseJSON:    false,
		Retry: retryConfig{
			MaxAttempts:            3,
			InitialBackOffDuration: time.Second,
			MaxBackOffDuration:     30 * time.Second,
			BackOffMultiplier:      2,
		},
	}
}
