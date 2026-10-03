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
<<<<<<< HEAD
=======
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-ldap/ldap/v3"
>>>>>>> b670dd6 (translate_ldap_attribute: fix String race during LDAP init (#51690))

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
<<<<<<< HEAD
=======

	// description stores the processor's String output. It must remain lock-free because the logger
	// evaluates this Stringer while client initialization is running.
	description atomic.Value // stores string

	clientMu          sync.Mutex
	clientErr         error
	nextClientAttempt time.Time
>>>>>>> b670dd6 (translate_ldap_attribute: fix String race during LDAP init (#51690))
}

func New(cfg *conf.C, log *logp.Logger) (beat.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("fail to unpack the translate_ldap_attribute configuration: %w", err)
	}

	return newFromConfig(c, log)
}

func newFromConfig(c config, logger *logp.Logger) (*processor, error) {
<<<<<<< HEAD
=======
	p := &processor{config: c}
	p.storeDescription(c)
	p.log = logger.Named(logName).With(logp.Stringer("processor", p))
	return p, nil
}

func (p *processor) storeDescription(c config) {
	p.description.Store(fmt.Sprintf("translate_ldap_attribute=[field=%s, ldap_address=%s, ldap_base_dn=%s, ldap_bind_user=%s, ldap_search_attribute=%s, ldap_mapped_attribute=%s]",
		c.Field, c.LDAPAddress, c.LDAPBaseDN, c.LDAPBindUser, c.LDAPSearchAttribute, c.LDAPMappedAttribute))
}

// newClient creates a new LDAP client by discovering and connecting to available servers.
func newClient(c config, log *logp.Logger) (*ldapClient, error) {
	// Auto-discover LDAP addresses if not provided
	var addresses []string
	if c.LDAPAddress != "" {
		addresses = []string{c.LDAPAddress}
	} else {
		log.Info("LDAP address not configured, attempting auto-discovery")
		discoveredAddresses, err := discoverLDAPAddress(c.LDAPDomain, log)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-discover LDAP server: %w", err)
		}
		addresses = discoveredAddresses
		log.Infow("discovered LDAP servers", "count", len(addresses), "addresses", addresses)
	}

	// Prepare base LDAP config
>>>>>>> b670dd6 (translate_ldap_attribute: fix String race during LDAP init (#51690))
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
	description, _ := p.description.Load().(string)
	return description
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

	guidString, ok := v.(string)
	if !ok {
		return errInvalidType
	}

	p.log.Debugw("ldap search", "guid", guidString)
	cn, err := p.client.findObjectBy(guidString)
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

func (p *processor) Close() error {
	p.client.close()
	return nil
}
<<<<<<< HEAD
=======

func (p *processor) ensureClient() (*ldapClient, error) {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	if p.client != nil {
		return p.client, nil
	}

	now := time.Now()
	if !p.nextClientAttempt.IsZero() && now.Before(p.nextClientAttempt) && p.clientErr != nil {
		return nil, fmt.Errorf("ldap client initialization paused until %s: %w", p.nextClientAttempt.Format(time.RFC3339), p.clientErr)
	}

	client, err := newClient(p.config, p.log)
	if err != nil {
		p.clientErr = err
		p.nextClientAttempt = now.Add(clientRetryBackoff)
		return nil, err
	}

	// Update config with discovered values for logging/debugging.
	p.client = client
	p.LDAPBaseDN = client.baseDN
	p.LDAPAddress = client.address
	p.storeDescription(p.config)
	p.clientErr = nil
	p.nextClientAttempt = time.Time{}
	return client, nil
}
>>>>>>> b670dd6 (translate_ldap_attribute: fix String race during LDAP init (#51690))
