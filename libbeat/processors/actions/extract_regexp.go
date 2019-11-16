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

package actions

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
)

type extractRegexp struct {
	Field     string
	Prefix    string
	RegexpStr string
	Regexp    *regexp.Regexp
}

func init() {
	processors.RegisterPlugin("extract_regexp",
		checks.ConfigChecked(NewExtractRegexp,
			checks.RequireFields("field", "regexp"),
			checks.AllowedFields("field", "regexp", "prefix")))
}

// NewExtractRegexp returns a new extract regexp processor. This gets a regexp, an event field
// and a prefix and finds all the expression defined names on the field adding them on the event
// pefixed by the defined prefix.
func NewExtractRegexp(c *common.Config) (processors.Processor, error) {
	config := struct {
		Regexp string `config:"regexp"`
		Field  string `config:"field"`
		Prefix string `config:"prefix"`
	}{}
	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the extract_regexp configuration: %s", err)
	}

	/* remove read only fields */
	for _, readOnly := range processors.MandatoryExportedFields {
		if config.Field == readOnly {
			return nil, fmt.Errorf("%s is a read only field, cannot override", readOnly)
		}
	}

	r, err := regexp.Compile(config.Regexp)
	if err != nil {
		return nil, fmt.Errorf("fail to compile regexp: %s", err)
	}

	f := &extractRegexp{
		Regexp:    r,
		RegexpStr: config.Regexp,
		Field:     config.Field,
		Prefix:    config.Prefix,
	}
	return f, nil
}

func (f *extractRegexp) Run(event *beat.Event) (*beat.Event, error) {
	fieldValue, err := event.GetValue(f.Field)
	if err != nil {
		return event, fmt.Errorf("error getting field '%s' from event", f.Field)
	}

	value, ok := fieldValue.(string)
	if !ok {
		return event, fmt.Errorf("could not get a string from field '%s'", f.Field)
	}

	matchs := f.Regexp.FindStringSubmatch(value)
	for i, name := range f.Regexp.SubexpNames() {
		if i != 0 && name != "" {
			event.PutValue(f.Prefix+name, matchs[i])
		}
	}
	return event, nil
}

func (f extractRegexp) String() string {
	return "extract_regexp=" + f.RegexpStr
}
