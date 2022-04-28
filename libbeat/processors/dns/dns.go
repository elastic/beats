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
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const logName = "processor.dns"

// instanceID is used to assign each instance a unique monitoring namespace.
var instanceID = atomic.MakeUint32(0)

func init() {
	processors.RegisterPlugin("dns", New)
	jsprocessor.RegisterPlugin("DNS", New)
}

type processor struct {
	Config
	resolver PTRResolver
	log      *logp.Logger
}

// New constructs a new DNS processor.
func New(cfg *common.Config) (processors.Processor, error) {
	c := defaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the dns configuration")
	}

	// Logging and metrics (each processor instance has a unique ID).
	var (
		id      = int(instanceID.Inc())
		log     = logp.NewLogger(logName).With("instance_id", id)
		metrics = monitoring.Default.NewRegistry(logName+"."+strconv.Itoa(id), monitoring.DoNotReport)
	)

	log.Debugf("DNS processor config: %+v", c)
	resolver, err := NewMiekgResolver(metrics, c.Timeout, c.Transport, c.Nameservers...)
	if err != nil {
		return nil, err
	}

	cache, err := NewPTRLookupCache(metrics.NewRegistry("cache"), c.CacheConfig, resolver)
	if err != nil {
		return nil, err
	}

	return &processor{Config: c, resolver: cache, log: log}, nil
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	var tagOnce sync.Once
	for field, target := range p.reverseFlat {
		if err := p.processField(field, target, p.Action, event); err != nil {
			p.log.Debugf("DNS processor failed: %v", err)
			tagOnce.Do(func() { mapstr.AddTags(event.Fields, p.TagOnFailure) })
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

	ptrRecord, err := p.resolver.LookupPTR(maybeIP)
	if err != nil {
		return fmt.Errorf("reverse lookup of %v value '%v' failed: %v", source, maybeIP, err)
	}

	return setFieldValue(action, event, target, ptrRecord.Host)
}

func setFieldValue(action FieldAction, event *beat.Event, key string, value string) error {
	switch action {
	case ActionReplace:
		_, err := event.PutValue(key, value)
		return err
	case ActionAppend:
		old, err := event.PutValue(key, value)
		if err != nil {
			return err
		}

		if old != nil {
			switch v := old.(type) {
			case string:
				_, err = event.PutValue(key, []string{v, value})
			case []string:
				_, err = event.PutValue(key, append(v, value))
			}
		}
		return err
	default:
		panic(errors.Errorf("Unexpected dns field action value encountered: %v", action))
	}
}

func (p processor) String() string {
	return fmt.Sprintf("dns=[timeout=%v, nameservers=[%v], action=%v, type=%v, fields=[%+v]",
		p.Timeout, strings.Join(p.Nameservers, ","), p.Action, p.Type, p.reverseFlat)
}
