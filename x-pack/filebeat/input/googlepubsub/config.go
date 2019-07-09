// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlepubsub

import "github.com/pkg/errors"

type config struct {
	// Google Cloud project name.
	ProjectID string `config:"project_id" validate:"required"`

	// Google Cloud Pub/Sub topic name.
	Topic string `config:"topic" validate:"required"`

	// Google Cloud Pub/Sub subscription name. Multiple Filebeats can pull from same subscription.
	Subscription struct {
		Name                   string `config:"name" validate:"required"`
		NumGoroutines          int    `config:"num_goroutines"`
		MaxOutstandingMessages int    `config:"max_outstanding_messages"`
		Create                 bool   `config:"create"`
	} `config:"subscription"`

	// JSON file containing authentication credentials and key.
	CredentialsFile string `config:"credentials_file"`

	// JSON blob containing authentication credentials and key.
	CredentialsJSON []byte `config:"credentials_json"`
}

func (c *config) Validate() error {
	if c.CredentialsFile == "" && len(c.CredentialsJSON) == 0 {
		return errors.New("credentials_file or credentials_json is required for pubsub input")
	}
	return nil
}

func defaultConfig() config {
	var c config
	c.Subscription.NumGoroutines = 1
	c.Subscription.MaxOutstandingMessages = 1000
	c.Subscription.Create = true
	return c
}
