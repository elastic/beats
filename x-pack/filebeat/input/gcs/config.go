// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/match"
)

// MaxWorkers - Defines the maximum number of go routines that will be spawned.
// Poll - Defines if polling should be performed on the input bucket source.
// PollInterval - Defines the maximum amount of time to wait before polling for the
// next batch of objects from the bucket.
// BucketTimeOut - Defines the maximum time that the sdk will wait for a bucket api response before timing out.
// ParseJSON - Informs the publisher whether to parse & objectify json data or not. By default this is set to
// false, since it can get expensive dealing with highly nested json data.
// FileSelectors - Defines a list of regex patterns that can be used to filter out objects from the bucket.
// TimeStampEpoch - Defines the epoch time in seconds, which is used to filter out objects that are older than the specified timestamp.
// ExpandEventListFromField - Defines the field name that will be used to expand the event into separate events.
// MaxWorkers, Poll, PollInterval, BucketTimeOut, ParseJSON, FileSelectors, TimeStampEpoch & ExpandEventListFromField
// can be configured at a global level, which applies to all buckets, as well as at the bucket level.
// Bucket level configurations will always override global level values.
type config struct {
	ProjectId                string               `config:"project_id" validate:"required"`
	Auth                     authConfig           `config:"auth" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers,omitempty" validate:"max=5000"`
	Poll                     *bool                `config:"poll,omitempty"`
	PollInterval             *time.Duration       `config:"poll_interval,omitempty"`
	ParseJSON                *bool                `config:"parse_json,omitempty"`
	BucketTimeOut            *time.Duration       `config:"bucket_timeout,omitempty"`
	Buckets                  []bucket             `config:"buckets" validate:"required"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
	// This field is only used for system test purposes, to override the HTTP endpoint.
	AlternativeHost string `config:"alternative_host,omitempty"`
}

// bucket contains the config for each specific object storage bucket in the root account
type bucket struct {
	Name                     string               `config:"name" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers,omitempty" validate:"max=5000"`
	BucketTimeOut            *time.Duration       `config:"bucket_timeout,omitempty"`
	Poll                     *bool                `config:"poll,omitempty"`
	PollInterval             *time.Duration       `config:"poll_interval,omitempty"`
	ParseJSON                *bool                `config:"parse_json,omitempty"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// fileSelectorConfig helps filter out gcs objects based on a regex pattern
type fileSelectorConfig struct {
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
}

type authConfig struct {
	CredentialsJSON *jsonCredentialsConfig `config:"credentials_json,omitempty"`
	CredentialsFile *fileCredentialsConfig `config:"credentials_file,omitempty"`
}

type fileCredentialsConfig struct {
	Path string `config:"path,omitempty"`
}
type jsonCredentialsConfig struct {
	AccountKey string `config:"account_key"`
}
