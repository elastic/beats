// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kerberos

import (
	"errors"
	"fmt"
)

type AuthType uint

const (
	authPassword = 1
	authKeytab   = 2

	authPasswordStr = "password"
	authKeytabStr   = "keytab"
)

var (
	InvalidAuthType = errors.New("invalid authentication type")

	authTypes = map[string]AuthType{
		authPasswordStr: authPassword,
		authKeytabStr:   authKeytab,
	}
)

type Config struct {
	Enabled     *bool    `config:"enabled" yaml:"enabled,omitempty"`
	AuthType    AuthType `config:"auth_type" validate:"required"`
	KeyTabPath  string   `config:"keytab"`
	ConfigPath  string   `config:"config_path" validate:"required"`
	ServiceName string   `config:"service_name"`
	Username    string   `config:"username"`
	Password    string   `config:"password"`
	Realm       string   `config:"realm" validate:"required"`
	EnableFAST  bool     `config:"enable_krb5_fast"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *Config) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

// Unpack validates and unpack "auth_type" config option
func (t *AuthType) Unpack(value string) error {
	authT, ok := authTypes[value]
	if !ok {
		return fmt.Errorf("invalid authentication type '%s'", value)
	}

	*t = authT

	return nil
}

func (c *Config) Validate() error {
	switch c.AuthType {
	case authPassword:
		if c.Username == "" {
			return fmt.Errorf("password authentication is selected for Kerberos, but username is not configured")
		}
		if c.Password == "" {
			return fmt.Errorf("password authentication is selected for Kerberos, but password is not configured")
		}

	case authKeytab:
		if c.KeyTabPath == "" {
			return fmt.Errorf("keytab authentication is selected for Kerberos, but path to keytab is not configured")
		}
	default:
		return InvalidAuthType
	}

	return nil
}
