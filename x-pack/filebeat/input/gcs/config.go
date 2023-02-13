// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"time"
)

// MaxWorkers - Defines the maximum number of go routines that will be spawned.
// Poll - Defines if polling should be performed on the input bucket source.
// PollInterval - Defines the maximum amount of time to wait before polling for the
// next batch of objects from the bucket.
// BucketTimeOut - Defines the maximum time that the sdk will wait for a bucket api response before timing out.
// ParseJSON - Informs the publisher whether to parse & objectify json data or not. By default this is set to
// false, since it can get expensive dealing with highly nested json data.
// MaxWorkers, Poll, PollInterval, BucketTimeOut, ParseJSON can be configured at a global level,
// which applies to all buckets, as well as at the bucket level.
// Bucket level configurations will always override global level values.
type config struct {
	ProjectId     string         `config:"project_id" validate:"required"`
	Auth          authConfig     `config:"auth" validate:"required"`
	MaxWorkers    *int           `config:"max_workers,omitempty" validate:"max=5000"`
	Poll          *bool          `config:"poll,omitempty"`
	PollInterval  *time.Duration `config:"poll_interval,omitempty"`
	ParseJSON     *bool          `config:"parse_json,omitempty"`
	BucketTimeOut *time.Duration `config:"bucket_timeout,omitempty"`
	Buckets       []bucket       `config:"buckets" validate:"required"`
	// This field is only used for system test purposes, to override the HTTP endpoint.
	AlternativeHost string `config:"alternative_host,omitempty"`
}

// bucket contains the config for each specific object storage bucket in the root account
type bucket struct {
	Name          string         `config:"name" validate:"required"`
	MaxWorkers    *int           `config:"max_workers,omitempty" validate:"max=5000"`
	BucketTimeOut *time.Duration `config:"bucket_timeout,omitempty"`
	Poll          *bool          `config:"poll,omitempty"`
	PollInterval  *time.Duration `config:"poll_interval,omitempty"`
	ParseJSON     *bool          `config:"parse_json,omitempty"`
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
