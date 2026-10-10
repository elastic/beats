// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"errors"
	"fmt"
	"net/url"
)

const (
	defaultKVMount      = "secret"
	defaultAppRoleMount = "approle"
)

// config is one "vault" entry of a monitor's secret_stores list.
type config struct {
	Address   string     `config:"address"`
	Namespace string     `config:"namespace"`
	KVMount   string     `config:"kv_mount"`
	Auth      authConfig `config:"auth"`
}

// authConfig holds exactly one authentication method.
type authConfig struct {
	Token   *tokenConfig   `config:"token"`
	AppRole *appRoleConfig `config:"approle"`
}

type tokenConfig struct {
	Value string `config:"value"`
}

type appRoleConfig struct {
	MountPath string `config:"mount_path"`
	RoleID    string `config:"role_id"`
	SecretID  string `config:"secret_id"`
}

func defaultConfig() config {
	return config{KVMount: defaultKVMount}
}

// validate checks the config and applies defaults that depend on the
// authentication method.
func (c *config) validate() error {
	if c.Address == "" {
		return errors.New("'address' is required")
	}
	u, err := url.Parse(c.Address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("'address' must be an http or https URL, got %q", c.Address)
	}
	if c.KVMount == "" {
		return errors.New("'kv_mount' must not be empty")
	}

	switch {
	case c.Auth.Token != nil && c.Auth.AppRole != nil:
		return errors.New("'auth' must have exactly one method, got both 'token' and 'approle'")
	case c.Auth.Token != nil:
		if c.Auth.Token.Value == "" {
			return errors.New("'auth.token.value' is required")
		}
	case c.Auth.AppRole != nil:
		if c.Auth.AppRole.RoleID == "" || c.Auth.AppRole.SecretID == "" {
			return errors.New("'auth.approle.role_id' and 'auth.approle.secret_id' are required")
		}
		if c.Auth.AppRole.MountPath == "" {
			c.Auth.AppRole.MountPath = defaultAppRoleMount
		}
	default:
		return errors.New("'auth' must have one method: 'token' or 'approle'")
	}
	return nil
}
