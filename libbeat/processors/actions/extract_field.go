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
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type extract_field struct {
	Field     string
	Separator string
	Index     int
	Target    string
}

/*
This one won't be registered (yet)

func init() {
	processors.RegisterPlugin("extract_field",
		configChecked(NewExtractField,
			requireFields("field", "separator", "index", "target"),
			allowedFields("field", "separator", "index", "target", "when")))
}
*/

func NewExtractField(c *conf.C) (processors.Processor, error) {
	config := struct {
		Field     string `config:"field"`
		Separator string `config:"separator"`
		Index     int    `config:"index"`
		Target    string `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the extract_field configuration: %s", err)
	}

	/* remove read only fields */
	for _, readOnly := range processors.MandatoryExportedFields {
		if config.Field == readOnly {
			return nil, fmt.Errorf("%s is a read only field, cannot override", readOnly)
		}
	}

	f := &extract_field{
		Field:     config.Field,
		Separator: config.Separator,
		Index:     config.Index,
		Target:    config.Target,
	}
	return f, nil
}

func (f *extract_field) Run(event *beat.Event) (*beat.Event, error) {
	fieldValue, err := event.GetValue(f.Field)
	if err != nil {
		return event, fmt.Errorf("error getting field '%s' from event", f.Field)
	}

	value, ok := fieldValue.(string)
	if !ok {
		return event, fmt.Errorf("could not get a string from field '%s'", f.Field)
	}

	parts := strings.Split(value, f.Separator)
	parts = deleteEmpty(parts)
	if len(parts) < f.Index+1 {
		return event, fmt.Errorf("index is out of range for field '%s'", f.Field)
	}

	event.PutValue(f.Target, parts[f.Index])

	return event, nil
}

func (f extract_field) String() string {
	return "extract_field=" + f.Target
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
