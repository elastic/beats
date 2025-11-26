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
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const logName = "processor.translate_ldap_attribute"

var errInvalidType = errors.New("search attribute field value is not a string")

func init() {
	processors.RegisterPlugin("translate_ldap_attribute", New)
	jsprocessor.RegisterPlugin("TranslateLDAPAttribute", New)
}

type processor struct {
	config
	client *ldapClient
	log    *logp.Logger
}

func New(cfg *conf.C, log *logp.Logger) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the translate_ldap_attribute configuration: %w", err)
	}

	return newFromConfig(c, log)
}

func newFromConfig(c config, logger *logp.Logger) (*processor, error) {
	ldapConfig := &ldapConfig{
		address:         c.LDAPAddress,
		baseDN:          c.LDAPBaseDN,
		username:        c.LDAPBindUser,
		password:        c.LDAPBindPassword,
		searchAttr:      c.LDAPSearchAttribute,
		mappedAttr:      c.LDAPMappedAttribute,
		searchTimeLimit: c.LDAPSearchTimeLimit,
	}
	if c.LDAPTLS != nil {
		tlsConfig, err := tlscommon.LoadTLSConfig(c.LDAPTLS, logger)
		if err != nil {
			return nil, fmt.Errorf("could not load provided LDAP TLS configuration: %w", err)
		}
		ldapConfig.tlsConfig = tlsConfig.ToConfig()
	}
	p := &processor{config: c}
	p.log = logger.Named(logName).With(logp.Stringer("processor", p))
	client, err := newLDAPClient(ldapConfig, p.log)
	if err != nil {
		return nil, err
	}
	p.client = client
	return p, nil
}

func (p *processor) String() string {
	return fmt.Sprintf("translate_ldap_attribute=[field=%s, ldap_address=%s, ldap_base_dn=%s, ldap_bind_user=%s, ldap_search_attribute=%s, ldap_mapped_attribute=%s]",
		p.Field, p.LDAPAddress, p.LDAPBaseDN, p.LDAPBindUser, p.LDAPSearchAttribute, p.LDAPMappedAttribute)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	p.log.Debugw("run ldap translation")
	err := p.translateLDAPAttr(event)
	if err != nil {
		// Always log errors at debug level, even when we are
		// ignoring failures.
		p.log.Debugw("ldap translation error", "error", err)
	} else {
		p.log.Debugw("ldap translation complete")
	}
	if err == nil || p.IgnoreFailure || (p.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound)) {
		return event, nil
	}
	return event, err
}

func (p *processor) translateLDAPAttr(event *beat.Event) error {
	v, err := event.GetValue(p.Field)
	if err != nil {
		return err
	}

	searchValue, ok := v.(string)
	if !ok {
		return errInvalidType
	}

	searchFilter, err := p.prepareSearchFilter(searchValue)
	if err != nil {
		return err
	}

	p.log.Debugw("ldap search", "search_value", searchValue, "filter_value", searchFilter)
	cn, err := p.client.findObjectBy(searchFilter)
	p.log.Debugw("ldap result", "common_name", cn)
	if err != nil {
		return err
	}

	field := p.Field
	if p.TargetField != "" {
		field = p.TargetField
	}
	_, err = event.PutValue(field, cn)
	return err
}

// prepareSearchFilter converts the search value to the appropriate format for LDAP queries.
// It applies GUID binary conversion when required based on the ADGUIDTranslation configuration.
func (p *processor) prepareSearchFilter(searchValue string) (string, error) {
	// Determine if GUID conversion should be applied
	var shouldConvertGUID bool
	if p.ADGUIDTranslation == nil {
		shouldConvertGUID = (p.LDAPSearchAttribute == "objectGUID")
	} else {
		shouldConvertGUID = *p.ADGUIDTranslation
	}

	if !shouldConvertGUID {
		return searchValue, nil
	}

	guidBytes, err := guidToBytes(searchValue)
	if err != nil {
		return "", fmt.Errorf("failed to convert GUID: %w", err)
	}
	searchFilter := escapeBinaryForLDAP(guidBytes)
	return searchFilter, nil
}

func (p *processor) Close() error {
	p.client.close()
	return nil
}
