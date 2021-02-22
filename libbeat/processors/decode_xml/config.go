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

package decode_xml

type decodeXMLConfig struct {
	Field         string  `config:"field" validate:"required"`
	Target        *string `config:"target_field"`
	OverwriteKeys bool    `config:"overwrite_keys"`
	DocumentID    string  `config:"document_id"`
	ToLower       bool    `config:"to_lower"`
	IgnoreMissing bool    `config:"ignore_missing"`
	IgnoreFailure bool    `config:"ignore_failure"`
}

func defaultConfig() decodeXMLConfig {
	return decodeXMLConfig{
		Field:         "message",
		OverwriteKeys: true,
		ToLower:       true,
	}
}
