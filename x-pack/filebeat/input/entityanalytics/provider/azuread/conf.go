// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"errors"
	"time"
)

const (
	// The default incremental update interval.
	defaultUpdateInterval = time.Minute * 15
	// The default full synchronization interval.
	defaultSyncInterval = time.Hour * 24
)

// conf contains parameters needed to configure the input.
type conf struct {
	TenantID       string        `config:"tenant_id" validate:"required"`
	SyncInterval   time.Duration `config:"sync_interval"`
	UpdateInterval time.Duration `config:"update_interval"`
}

// Validate runs validation against the config.
func (c *conf) Validate() error {
	if c.SyncInterval < c.UpdateInterval {
		return errors.New("sync_interval must be longer than update_interval")
	}
	if c.SyncInterval == 0 {
		return errors.New("sync_interval must not be zero")
	}
	if c.UpdateInterval == 0 {
		return errors.New("update_interval must not be zero")
	}

	return nil
}

// defaultConfig returns a default configuration.
func defaultConf() conf {
	return conf{
		SyncInterval:   defaultSyncInterval,
		UpdateInterval: defaultUpdateInterval,
	}
}
