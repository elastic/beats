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
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	guidTranslationAuto   = "auto"
	guidTranslationAlways = "always"
	guidTranslationNever  = "never"
)

type config struct {
	Field               string            `config:"field"  validate:"required"`
	TargetField         string            `config:"target_field"`
	LDAPAddress         string            `config:"ldap_address"`
	LDAPBaseDN          string            `config:"ldap_base_dn"`
	LDAPBindUser        string            `config:"ldap_bind_user"`
	LDAPBindPassword    string            `config:"ldap_bind_password"`
	LDAPSearchAttribute string            `config:"ldap_search_attribute" validate:"required"`
	LDAPMappedAttribute string            `config:"ldap_mapped_attribute" validate:"required"`
	LDAPSearchTimeLimit int               `config:"ldap_search_time_limit"`
	LDAPTLS             *tlscommon.Config `config:"ldap_ssl"`

	// ADGUIDTranslation controls Active Directory GUID binary conversion.
	// Supported values:
	//   "auto"   (default): Enable GUID conversion when objectGUID is used against AD
	//   "always": Always apply GUID conversion regardless of attribute name
	//   "never" : Never apply GUID conversion
	ADGUIDTranslation string `config:"ad_guid_translation"`

	IgnoreMissing bool `config:"ignore_missing"`
	IgnoreFailure bool `config:"ignore_failure"`
}

func defaultConfig() config {
	return config{
		LDAPSearchAttribute: "objectGUID",
		LDAPMappedAttribute: "cn",
		LDAPSearchTimeLimit: 30,
		ADGUIDTranslation:   guidTranslationAuto,
	}
}

func (c *config) validate() error {
	switch strings.ToLower(strings.TrimSpace(c.ADGUIDTranslation)) {
	case "", guidTranslationAuto:
		c.ADGUIDTranslation = guidTranslationAuto
	case guidTranslationAlways:
		c.ADGUIDTranslation = guidTranslationAlways
	case guidTranslationNever:
		c.ADGUIDTranslation = guidTranslationNever
	default:
		return fmt.Errorf("invalid ad_guid_translation value %q (expected auto|always|never)", c.ADGUIDTranslation)
	}
	return nil
}
