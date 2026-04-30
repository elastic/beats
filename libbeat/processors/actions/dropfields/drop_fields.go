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

package dropfields

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/match"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type dropFields struct {
	Fields        []string
	RegexpFields  []match.Matcher
	IgnoreMissing bool
	Cleanup       bool
}

func NewDropFields(c *conf.C, log *logp.Logger) (beat.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		Cleanup       bool     `config:"cleanup"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the drop_fields configuration: %w", err)
	}

	// Do not drop manadatory fields
	var configFields []string
	for _, readOnly := range processors.MandatoryExportedFields {
		for _, field := range config.Fields {
			if readOnly == field || strings.HasPrefix(field, readOnly+".") {
				continue
			}
			configFields = append(configFields, field)
		}
	}

	// Parse regexp containing fields and removes them from initial config
	regexpFields := make([]match.Matcher, 0)
	for i := len(configFields) - 1; i >= 0; i-- {
		field := configFields[i]
		if strings.HasPrefix(field, "/") && strings.HasSuffix(field, "/") && len(field) > 2 {
			configFields = append(configFields[:i], configFields[i+1:]...)

			matcher, err := match.Compile(field[1 : len(field)-1])
			if err != nil {
				return nil, fmt.Errorf("wrong configuration in drop_fields[%d]=%s. %w", i, field, err)
			}

			regexpFields = append(regexpFields, matcher)
		}
	}

	f := &dropFields{Fields: configFields, IgnoreMissing: config.IgnoreMissing, RegexpFields: regexpFields, Cleanup: config.Cleanup}
	return f, nil
}

func (f *dropFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []error

	// remove exact match fields
	for _, field := range f.Fields {
		f.deleteField(event, field, &errs)
	}

	// remove fields contained in regexp expressions
	for _, regex := range f.RegexpFields {
		for _, field := range *event.Fields.FlattenKeys() {
			if regex.MatchString(field) {
				f.deleteField(event, field, &errs)
			}
		}
	}

	return event, errors.Join(errs...)
}

func (f *dropFields) deleteField(event *beat.Event, field string, errs *[]error) {
	var err error
	if f.Cleanup {
		err = event.DeleteWithCleanup(field)
	} else {
		err = event.Delete(field)
	}
	if err != nil {
		if !f.IgnoreMissing || !errors.Is(err, mapstr.ErrKeyNotFound) {
			*errs = append(*errs, fmt.Errorf("failed to drop field [%v], error: %w", field, err))
		}
	}
}

func (f *dropFields) String() string {
	json, _ := json.Marshal(f)
	return "drop_fields=" + string(json)
}
