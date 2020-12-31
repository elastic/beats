// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

// Config exposes the configuration options of the GCP provider.
type Config struct {
	Location        string `config:"location_id" validate:"required"`
	ProjectID       string `config:"project_id" validate:"required"`
	FunctionStorage string `config:"storage_name" validate:"required"`
}
