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
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func init() {
	processors.RegisterPlugin(
		"uppercase",
		checks.ConfigChecked(
			NewUpperCaseProcessor,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "fail_on_error", "alter_full_field", "values"),
		),
	)
}

// NewUpperCaseProcessor converts event keys matching the provided fields to uppercase
func NewUpperCaseProcessor(c *conf.C) (beat.Processor, error) {
	return NewAlterFieldProcessor(c, "uppercase", upperCase)
}

func upperCase(field string) (string, error) {
	return strings.ToUpper(field), nil
}
