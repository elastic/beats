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

//go:build !requirefips

package kerberos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	cfg := &Config{
		AuthType:   authPassword,
		Username:   "username",
		Password:   "password",
		ConfigPath: "/etc/krb5.conf",
	}
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfigValidateKrb5Source(t *testing.T) {
	base := Config{
		AuthType: authPassword,
		Username: "username",
		Password: "password",
		Realm:    "EXAMPLE.COM",
	}

	pathOnly := base
	pathOnly.ConfigPath = "/etc/krb5.conf"
	require.NoError(t, pathOnly.Validate(), "config_path alone should be accepted")

	inline := base
	inline.Krb5Conf = "[libdefaults]\n  default_realm = EXAMPLE.COM\n"
	require.NoError(t, inline.Validate(), "inline krb5_conf alone should be accepted")

	neither := base
	err := neither.Validate()
	require.Error(t, err, "kerberos must name a krb5 config source")
	assert.Contains(t, err.Error(), "config_path", "error should name the file option")
	assert.Contains(t, err.Error(), "krb5_conf", "error should name the inline option")

	both := base
	both.ConfigPath = "/etc/krb5.conf"
	both.Krb5Conf = "[libdefaults]\n"
	err = both.Validate()
	require.Error(t, err, "config_path and krb5_conf together are ambiguous")
	assert.Contains(t, err.Error(), "exactly one")
}
