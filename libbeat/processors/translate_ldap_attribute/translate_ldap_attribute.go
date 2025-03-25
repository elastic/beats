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

func New(cfg *conf.C) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the translate_ldap_attribute configuration: %w", err)
	}

	return newFromConfig(c)
}

func newFromConfig(c config) (*processor, error) {
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
		tlsConfig, err := tlscommon.LoadTLSConfig(c.LDAPTLS)
		if err != nil {
			return nil, fmt.Errorf("could not load provided LDAP TLS configuration: %w", err)
		}
		ldapConfig.tlsConfig = tlsConfig.ToConfig()
	}
	client, err := newLDAPClient(ldapConfig)
	if err != nil {
		return nil, err
	}
	return &processor{
		config: c,
		client: client,
		log:    logp.NewLogger(logName),
	}, nil
}

func (p *processor) String() string {
	return fmt.Sprintf("translate_ldap_attribute=[field=%s, ldap_address=%s, ldap_base_dn=%s, ldap_bind_user=%s, ldap_search_attribute=%s, ldap_mapped_attribute=%s]",
		p.Field, p.LDAPAddress, p.LDAPBaseDN, p.LDAPBindUser, p.LDAPSearchAttribute, p.LDAPMappedAttribute)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	err := p.translateLDAPAttr(event)
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

	guidString, ok := v.(string)
	if !ok {
		return errInvalidType
	}

	cn, err := p.client.findObjectBy(guidString)
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

func (p *processor) Close() error {
	p.client.close()
	return nil
}
