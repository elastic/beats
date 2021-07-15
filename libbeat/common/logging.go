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

package common

import "strings"

var maskList = MakeStringSet(
	"password",
	"passphrase",
	"key_passphrase",
	"pass",
	"proxy_url",
	"url",
	"urls",
	"host",
	"hosts",
	"authorization",
	"proxy-authorization",
)

func applyLoggingMask(c interface{}) {
	switch cfg := c.(type) {
	case map[string]interface{}:
		for k, v := range cfg {
			if maskList.Has(strings.ToLower(k)) {
				if arr, ok := v.([]interface{}); ok {
					for i := range arr {
						arr[i] = "xxxxx"
					}
				} else {
					cfg[k] = "xxxxx"
				}
			} else {
				applyLoggingMask(v)
			}
		}

	case []interface{}:
		for _, elem := range cfg {
			applyLoggingMask(elem)
		}
	}
}
