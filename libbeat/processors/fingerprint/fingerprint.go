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

package fingerprint

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	procName = "fingerprint"
)

func init() {
	processors.RegisterPlugin(procName, New)
	jsprocessor.RegisterPlugin("Fingerprint", New)
}

type fingerprint struct {
	config Config
	fields []string
	hash   hashMethod
}

// New constructs a new fingerprint processor.
func New(cfg *config.C) (beat.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, makeErrConfigUnpack(err)
	}

	// The fields array must be sorted, to guarantee that we always
	// get the same hash for a similar set of configured keys.
	// The call `ToSlice` always returns a sorted slice.
	fields := common.MakeStringSet(config.Fields...).ToSlice()

	p := &fingerprint{
		config: config,
		hash:   config.Method.Hash,
		fields: fields,
	}

	return p, nil
}

// Run enriches the given event with a fingerprint.
func (p *fingerprint) Run(event *beat.Event) (*beat.Event, error) {
	hashFn := p.hash()

	if err := p.writeFields(hashFn, event); err != nil {
		return nil, makeErrComputeFingerprint(err)
	}

	encodedHash := p.config.Encoding.Encode(hashFn.Sum(nil))

	if _, err := event.PutValue(p.config.TargetField, encodedHash); err != nil {
		return nil, makeErrComputeFingerprint(err)
	}

	return event, nil
}

func (p *fingerprint) String() string {
	json, _ := json.Marshal(&p.config)
	return procName + "=" + string(json)
}

func (p *fingerprint) writeFields(to io.Writer, event *beat.Event) error {
	for _, k := range p.fields {
		v, err := event.GetValue(k)
		if err != nil {
			if p.config.IgnoreMissing {
				continue
			}
			return makeErrMissingField(k, err)
		}

		switch vv := v.(type) {
		case map[string]interface{}, []interface{}, mapstr.M:
			return makeErrNonScalarField(k)
		case time.Time:
			// Ensure we consistently hash times in UTC.
			v = vv.UTC()
		}

		fmt.Fprintf(to, "|%v|%v", k, v)
	}

	_, _ = io.WriteString(to, "|")
	return nil
}
