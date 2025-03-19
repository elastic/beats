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

package translate_ldap_attribute

import (
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type config struct {
	Field               string            `config:"field"  validate:"required"`
	TargetField         string            `config:"target_field"`
	LDAPAddress         string            `config:"ldap_address" validate:"required"`
	LDAPBaseDN          string            `config:"ldap_base_dn" validate:"required"`
	LDAPBindUser        string            `config:"ldap_bind_user"`
	LDAPBindPassword    string            `config:"ldap_bind_password"`
	LDAPSearchAttribute string            `config:"ldap_search_attribute" validate:"required"`
	LDAPMappedAttribute string            `config:"ldap_mapped_attribute" validate:"required"`
	LDAPSearchTimeLimit int               `config:"ldap_search_time_limit"`
	LDAPTLS             *tlscommon.Config `config:"ldap_ssl"`

	IgnoreMissing bool `config:"ignore_missing"`
	IgnoreFailure bool `config:"ignore_failure"`
}

func defaultConfig() config {
	return config{
		LDAPSearchAttribute: "objectGUID",
		LDAPMappedAttribute: "cn",
		LDAPSearchTimeLimit: 30}
}
