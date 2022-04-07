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

package timestamp

import "github.com/elastic/beats/v8/libbeat/common/cfgtype"

type config struct {
	Field          string            `config:"field" validate:"required"`   // Source field containing time time to be parsed.
	TargetField    string            `config:"target_field"`                // Target field for the parsed time value. The target value is always written as UTC. Defaults to @timestamp.
	Layouts        []string          `config:"layouts" validate:"required"` // Timestamp layouts that define the expected time value format.
	Timezone       *cfgtype.Timezone `config:"timezone"`                    // IANA time zone (e.g. America/New_York) or fixed offset to use when parsing a timestamp not containing a timezone.
	IgnoreMissing  bool              `config:"ignore_missing"`              // Ignore errors when the source field is missing.
	IgnoreFailure  bool              `config:"ignore_failure"`              // Ignore errors when parsing the timestamp.
	TestTimestamps []string          `config:"test"`                        // A list of timestamps that must parse successfully when loading the processor.
	ID             string            `config:"id"`                          // An identifier for this processor. Useful for debugging.
}

func defaultConfig() config {
	return config{
		TargetField: "@timestamp",
	}
}
