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

// Kerberos config unpacking is rejected in FIPS builds (libbeat's
// kerberos.Config.Validate), so these success-path cases only run without
// the requirefips tag.
//go:build !requirefips

package http

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestAuthUnpackKerberos(t *testing.T) {
	kerberosYAML := "enabled: true\nauth_type: password\nrealm: CORP.LOCAL\nconfig_path: /etc/krb5.conf\nusername: svc\npassword: secret\n"

	unpack := func(t *testing.T, yaml string) Config {
		t.Helper()
		cfg, err := conf.NewConfigWithYAML([]byte(yaml), "test")
		require.NoError(t, err)
		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c))
		return c
	}

	t.Run("nested object", func(t *testing.T) {
		c := unpack(t, "urls: [http://x]\nkerberos:\n  enabled: true\n  auth_type: password\n  realm: CORP.LOCAL\n  config_path: /etc/krb5.conf\n  username: svc\n  password: secret\n")
		require.True(t, c.Kerberos.IsEnabled())
		assert.Equal(t, "CORP.LOCAL", c.Kerberos.Realm)
		assert.Equal(t, "svc", c.Kerberos.Username)
	})

	t.Run("base64 string", func(t *testing.T) {
		c := unpack(t, "urls: [http://x]\nkerberos: "+base64.StdEncoding.EncodeToString([]byte(kerberosYAML))+"\n")
		require.True(t, c.Kerberos.IsEnabled())
		assert.Equal(t, "CORP.LOCAL", c.Kerberos.Realm)
		assert.Equal(t, "svc", c.Kerberos.Username)
	})

	t.Run("base64 json", func(t *testing.T) {
		c := unpack(t, "urls: [http://x]\nkerberos: "+base64.StdEncoding.EncodeToString([]byte(`{"enabled":true,"auth_type":"password","realm":"CORP.LOCAL","config_path":"/etc/krb5.conf","username":"svc","password":"secret"}`))+"\n")
		require.True(t, c.Kerberos.IsEnabled())
		assert.Equal(t, "CORP.LOCAL", c.Kerberos.Realm)
	})
}
