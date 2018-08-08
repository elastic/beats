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

package dns

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/processors"
)

const logName = "processor.dns"

// instanceID is used to assign each instance a unique monitoring namespace.
var instanceID = atomic.MakeUint32(0)

func init() {
	processors.RegisterPlugin("dns", newDNSProcessor)
}

type processor struct {
	Config
	resolver PTRResolver
	log      *logp.Logger
}

func newDNSProcessor(cfg *common.Config) (processors.Processor, error) {
	c := defaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the dns configuration")
	}

	log := logp.NewLogger(logName)
	reg := monitoring.Default.NewRegistry(logName+"."+strconv.Itoa(int(instanceID.Inc())), monitoring.DoNotReport)

	resolver, err := NewMiekgResolver(reg, c.Timeout, c.Nameservers...)
	if err != nil {
		return nil, err
	}

	log.Debugf("DNS processor config: %+v", c)
	return &processor{
		Config: c,
		resolver: NewPTRLookupCache(reg.NewRegistry("cache"), log,
			c.CacheConfig, resolver),
		log: log,
	}, nil
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	for _, l := range p.Lookup {
		for field, target := range l.reverseFlat {
			p.processField(field, target, l.Action, event)
		}
	}
	return event, nil
}

func (p *processor) processField(source, target string, action FieldAction, event *beat.Event) error {
	v, err := event.GetValue(source)
	if err != nil {
		return nil
	}

	maybeIP, ok := v.(string)
	if !ok {
		return nil
	}

	name, err := p.resolver.LookupPTR(maybeIP)
	if err != nil {
		return nil
	}

	old, err := event.PutValue(target, name.Host)
	if err != nil {
		return err
	}

	if action == ActionAppend && old != nil {
		switch v := old.(type) {
		case string:
			_, err = event.PutValue(target, []string{v, name.Host})
		case []string:
			_, err = event.PutValue(target, append(v, name.Host))
		}
	}
	return err
}

func (p processor) String() string {
	return fmt.Sprintf("dns=[timeout=%v, nameservers=[%v], lookup=[%+v]]", p.Timeout, p.Nameservers, p.Lookup)
}
