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
	"fmt"
	"sort"
	"strconv"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	jsprocessor "github.com/elastic/beats/libbeat/processors/script/javascript/module/processor"
	"github.com/pkg/errors"
)

func init() {
	processors.RegisterPlugin("fingerprint", New)
	jsprocessor.RegisterPlugin("Fingerprint", New)
}

const processorName = "fingerprint"

type fingerprint struct {
	config Config
}

// New constructs a new fingerprint processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack %v processor configuration", processorName)
	}

	sort.Strings(config.Fields)

	p := &fingerprint{
		config: config,
	}

	return p, nil
}

// Run enriches the given event with fingerprint information
func (p *fingerprint) Run(event *beat.Event) (*beat.Event, error) {
	var str string
	for _, k := range p.config.Fields {
		v, err := event.Fields.GetValue(k)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compute fingerprint")
		}

		var s string
		switch v := v.(type) {
		case map[string]interface{}, []interface{}:
			return nil, errors.Errorf("cannot compute fingerprint using non-scalar field: %v", k)
		case string:
			s = v
		case int:
			s = strconv.Itoa(v)
		}

		str += fmt.Sprintf("|%v|%v", k, s)
	}
	str += "|"

	makeFingerprint, err := p.config.Method.factory()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute fingerprint")
	}

	f, err := makeFingerprint(str)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute fingerprint")
	}

	if _, err = event.PutValue(p.config.TargetField, f); err != nil {
		return nil, errors.Wrap(err, "failed to compute fingerprint")
	}

	return event, nil
}

func (p *fingerprint) String() string {
	return fmt.Sprintf("%v=[method=[%v]]", processorName, p.config.Method)
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
