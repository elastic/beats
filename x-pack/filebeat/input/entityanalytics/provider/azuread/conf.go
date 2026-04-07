// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"errors"
	"fmt"
	"strings"
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
	Dataset        string        `config:"dataset"`
	EnrichWith     []string      `config:"enrich_with"`
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
	switch strings.ToLower(c.Dataset) {
	case "", "all", "users", "devices":
	default:
		return errors.New("dataset must be 'all', 'users', 'devices' or empty")
	}

	for _, v := range c.EnrichWith {
		switch strings.ToLower(v) {
		case "mfa", "none":
		default:
			return fmt.Errorf("enrich_with value %q is not supported; valid values are 'mfa' and 'none'", v)
		}
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

func (c *conf) wantUsers() bool {
	switch strings.ToLower(c.Dataset) {
	case "", "all", "users":
		return true
	default:
		return false
	}
}

func (c *conf) wantDevices() bool {
	switch strings.ToLower(c.Dataset) {
	case "", "all", "devices":
		return true
	default:
		return false
	}
}

func (c *conf) wantMFA() bool {
	for _, v := range c.EnrichWith {
		if strings.ToLower(v) == "mfa" {
			return true
		}
	}
	return false
}
