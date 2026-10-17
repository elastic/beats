// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const defaultServicePath = "/AIMWebService/api/Accounts"

// config is one "cyberark" entry of a monitor's secret_stores list.
type config struct {
	Address     string            `config:"address"`
	AppID       string            `config:"app_id"`
	ServicePath string            `config:"service_path"`
	Auth        authConfig        `config:"auth"`
	SSL         *tlscommon.Config `config:"ssl"`
}

// authConfig holds optional HTTP authentication to the CyberArk Central
// Credential Provider (CCP). Mutual TLS is configured via ssl.certificate and
// ssl.key; IP-allowlisted CCP deployments need no auth block.
type authConfig struct {
	Token string       `config:"token"`
	Basic *basicConfig `config:"basic"`
}

type basicConfig struct {
	Username string `config:"username"`
	Password string `config:"password"`
}

func defaultConfig() config {
	return config{ServicePath: defaultServicePath}
}

func (c *config) validate() error {
	if c.Address == "" {
		return errors.New("'address' is required")
	}
	u, err := url.Parse(c.Address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("'address' must be an http or https URL, got %q", c.Address)
	}
	if strings.TrimSpace(c.AppID) == "" {
		return errors.New("'app_id' is required")
	}
	if strings.TrimSpace(c.ServicePath) == "" {
		return errors.New("'service_path' must not be empty")
	}
	if !strings.HasPrefix(c.ServicePath, "/") {
		c.ServicePath = "/" + c.ServicePath
	}

	if c.Auth.Token != "" && c.Auth.Basic != nil {
		return errors.New("'auth' must have at most one method, got both 'token' and 'basic'")
	}
	if c.Auth.Basic != nil && (c.Auth.Basic.Username == "" || c.Auth.Basic.Password == "") {
		return errors.New("'auth.basic.username' and 'auth.basic.password' are required")
	}
	return nil
}
