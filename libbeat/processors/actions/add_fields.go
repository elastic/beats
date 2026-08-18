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
	"github.com/elastic/beats/v9/libbeat/processors"
	"github.com/elastic/beats/v9/libbeat/processors/actions/addfields"
	"github.com/elastic/beats/v9/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v9/libbeat/processors/script/javascript/module/processor/registry"
)

func init() {
	processors.RegisterPlugin("add_fields",
		checks.ConfigChecked(addfields.CreateAddFields,
			checks.RequireFields(addfields.FieldsKey),
			checks.AllowedFields(addfields.FieldsKey, "target", "when")))

	jsprocessor.RegisterPlugin("AddFields", addfields.CreateAddFields)
}
