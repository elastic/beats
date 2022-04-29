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

//go:build windows
// +build windows

package translate_sid

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const logName = "processor.translate_sid"

var errInvalidType = errors.New("SID field value is not a string")

func init() {
	processors.RegisterPlugin("translate_sid", New)
	jsprocessor.RegisterPlugin("TranslateSID", New)
}

type processor struct {
	config
	log *logp.Logger
}

// New returns a new translate_sid processor for converting windows SID values
// to names.
func New(cfg *conf.C) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the translate_sid configuration")
	}

	return newFromConfig(c)
}

func newFromConfig(c config) (*processor, error) {
	return &processor{
		config: c,
		log:    logp.NewLogger(logName),
	}, nil
}

func (p *processor) String() string {
	return fmt.Sprintf("translate_sid=[field=%s, account_name_target=%s, account_type_target=%s, domain_target=%s]",
		p.Field, p.AccountNameTarget, p.AccountTypeTarget, p.DomainTarget)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	err := p.translateSID(event)
	if err == nil || p.IgnoreFailure || (p.IgnoreMissing && mapstr.ErrKeyNotFound == errors.Cause(err)) {
		return event, nil
	}
	return event, err
}

func (p *processor) translateSID(event *beat.Event) error {
	v, err := event.GetValue(p.Field)
	if err != nil {
		return err
	}
	sidString, ok := v.(string)
	if !ok {
		return errInvalidType
	}

	// All SIDs starting with S-1-15-3 are capability SIDs. Active Directory
	// does not resolve them so don't try.
	// Reference: https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
	if strings.HasPrefix(sidString, "S-1-15-3-") {
		return windows.ERROR_NONE_MAPPED

	}

	sid, err := windows.StringToSid(sidString)
	if err != nil {
		return err
	}

	// XXX: May want to introduce an in-memory cache if the lookups are time consuming.
	account, domain, accountType, err := sid.LookupAccount("")
	if err != nil {
		return err
	}

	// Do all operations even if one fails.
	var errs []error
	if p.AccountNameTarget != "" {
		if _, err = event.PutValue(p.AccountNameTarget, account); err != nil {
			errs = append(errs, err)
		}
	}
	if p.AccountTypeTarget != "" {
		if _, err = event.PutValue(p.AccountTypeTarget, winevent.SIDType(accountType).String()); err != nil {
			errs = append(errs, err)
		}
	}
	if p.DomainTarget != "" {
		if _, err = event.PutValue(p.DomainTarget, domain); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}
