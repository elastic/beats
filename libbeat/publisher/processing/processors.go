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

package processing

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type group struct {
	log   *logp.Logger
	title string
	list  []beat.Processor
}

type processorFn struct {
	name string
	fn   func(event *beat.Event) (*beat.Event, error)
}

func newGeneralizeProcessor(keepNull bool) *processorFn {
	logger := logp.NewLogger("publisher_processing")
	g := common.NewGenericEventConverter(keepNull)
	return newProcessor("generalizeEvent", func(event *beat.Event) (*beat.Event, error) {
		// Filter out empty events. Empty events are still reported by ACK callbacks.
		if len(event.Fields) == 0 {
			return nil, nil
		}

		// data streams require @timestamp field
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		fields := g.Convert(event.Fields)
		if fields == nil {
			logger.Error("fail to convert to generic event")
			return nil, nil
		}

		event.Fields = fields
		return event, nil
	})
}

var dropDisabledProcessor = newProcessor("dropDisabled", func(event *beat.Event) (*beat.Event, error) {
	return nil, nil
})

func newGroup(title string, log *logp.Logger) *group {
	return &group{
		title: title,
		log:   log,
	}
}

func (p *group) add(processor processors.Processor) {
	if processor != nil {
		p.list = append(p.list, processor)
	}
}

func (p *group) Close() error {
	if p == nil {
		return nil
	}
	var errs multierror.Errors
	for _, processor := range p.list {
		err := processors.Close(processor)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs.Err()
}

func (p *group) String() string {
	var s []string
	for _, p := range p.list {
		s = append(s, p.String())
	}

	str := strings.Join(s, ", ")
	if p.title == "" {
		return str
	}
	return fmt.Sprintf("%v{%v}", p.title, str)
}

func (p *group) All() []beat.Processor {
	return p.list
}

func (p *group) Run(event *beat.Event) (*beat.Event, error) {
	if p == nil || len(p.list) == 0 {
		return event, nil
	}

	for _, sub := range p.list {
		var err error

		event, err = sub.Run(event)
		if err != nil {
			// XXX: We don't drop the event, but continue filtering here if the most
			//      recent processor did return an event.
			//      We want processors having this kind of implicit behavior
			//      on errors?

			p.log.Debugf("Fail to apply processor %s: %s", p, err)
		}

		if event == nil {
			return nil, err
		}
	}

	return event, nil
}

func newProcessor(name string, fn func(*beat.Event) (*beat.Event, error)) *processorFn {
	return &processorFn{name: name, fn: fn}
}

func newAnnotateProcessor(name string, fn func(*beat.Event)) *processorFn {
	return newProcessor(name, func(event *beat.Event) (*beat.Event, error) {
		fn(event)
		return event, nil
	})
}

func (p *processorFn) String() string                         { return p.name }
func (p *processorFn) Run(e *beat.Event) (*beat.Event, error) { return p.fn(e) }

func clientEventMeta(meta mapstr.M, needsCopy bool) *processorFn {
	fn := func(event *beat.Event) { addMeta(event, meta) }
	if needsCopy {
		fn = func(event *beat.Event) { addMeta(event, meta.Clone()) }
	}
	return newAnnotateProcessor("@metadata", fn)
}

func addMeta(event *beat.Event, meta mapstr.M) {
	if event.Meta == nil {
		event.Meta = meta
	} else {
		event.Meta.Clone()
		event.Meta.DeepUpdate(meta)
	}
}

func makeAddDynMetaProcessor(
	name string,
	meta *mapstr.Pointer,
	checkCopy func(m mapstr.M) bool,
) *processorFn {
	return newAnnotateProcessor(name, func(event *beat.Event) {
		dynFields := meta.Get()
		if checkCopy(dynFields) {
			dynFields = dynFields.Clone()
		}

		event.Fields.DeepUpdate(dynFields)
	})
}

func debugPrintProcessor(info beat.Info, log *logp.Logger) *processorFn {
	// ensure only one go-routine is using the encoder (in case
	// beat.Client is shared between multiple go-routines by accident)
	var mux sync.Mutex

	encoder := json.New(info.Version, json.Config{
		Pretty:     true,
		EscapeHTML: false,
	})
	return newProcessor("debugPrint", func(event *beat.Event) (*beat.Event, error) {
		mux.Lock()
		defer mux.Unlock()

		b, err := encoder.Encode(info.Beat, event)
		if err != nil {
			return event, nil
		}

		log.Debugf("Publish event: %s", b)
		return event, nil
	})
}

func hasKey(m mapstr.M, key string) bool {
	_, exists := m[key]
	return exists
}

func hasKeyAnyOf(m, builtin mapstr.M) bool {
	for k := range builtin {
		if hasKey(m, k) {
			return true
		}
	}
	return false
}
