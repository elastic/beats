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

package decode_xml_wineventlog

type config struct {
	Field         string `config:"field" validate:"required"`
	Target        string `config:"target_field"`
	OverwriteKeys bool   `config:"overwrite_keys"`
	MapECSFields  bool   `config:"map_ecs_fields"`
	IgnoreMissing bool   `config:"ignore_missing"`
	IgnoreFailure bool   `config:"ignore_failure"`
}

func defaultConfig() config {
	return config{
		Field:         "message",
		OverwriteKeys: true,
		MapECSFields:  true,
		Target:        "winlog",
	}
}
