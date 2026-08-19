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

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/processors"
	"github.com/elastic/beats/v9/libbeat/processors/checks"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	processors.RegisterPlugin(
		"lowercase",
		checks.ConfigChecked(
			NewLowerCaseProcessor,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "fail_on_error", "alter_full_field", "values"),
		),
	)
}

// NewLowerCaseProcessor converts event keys matching the provided fields to lowercase
func NewLowerCaseProcessor(c *conf.C, log *logp.Logger) (beat.Processor, error) {
	return NewAlterFieldProcessor(c, "lowercase", lowerCase)
}

func lowerCase(field string) (string, error) {
	return strings.ToLower(field), nil
}
