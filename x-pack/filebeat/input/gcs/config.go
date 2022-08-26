// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package gcs

import (
	"time"
)

// MaxWorkers, Poll & PollInterval can be configured at a global level,
// which applies to all buckets , as well as at the container level.
// Container level configurations will always override global level values.
type config struct {
	ProjectId     string         `config:"project_id" validate:"required"`
	Auth          authConfig     `config:"auth" validate:"required"`
	MaxWorkers    *int           `config:"max_workers,omitempty" validate:"max=5000"`
	Poll          *bool          `config:"poll,omitempty"`
	PollInterval  *time.Duration `config:"poll_interval,omitempty"`
	BucketTimeOut *time.Duration `config:"bucket_timeout,omitempty"`
	Buckets       []bucket       `config:"buckets" validate:"required"`
}

// bucket contains the config for each specific object storage bucket in the root account
type bucket struct {
	Name          string         `config:"name" validate:"required"`
	MaxWorkers    *int           `config:"max_workers,omitempty" validate:"max=5000"`
	BucketTimeOut *time.Duration `config:"bucket_timeout,omitempty"`
	Poll          *bool          `config:"poll,omitempty"`
	PollInterval  *time.Duration `config:"poll_interval,omitempty"`
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

func defaultConfig() config {
	return config{
		ProjectId: "some_project",
	}
}
