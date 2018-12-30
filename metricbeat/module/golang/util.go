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

package golang

import (
	"bytes"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

/**
Convert cmd array to cmd line
*/
func GetCmdStr(v interface{}) interface{} {
	switch t := v.(type) {
	case []interface{}:
		var buffer bytes.Buffer
		strs := v.([]interface{})
		for _, v := range strs {
			buffer.WriteString(v.(string))
			buffer.WriteString(" ")
		}
		return strings.TrimRight(buffer.String(), " ")
	default:
		logp.Debug("golang", "unexpected cmdline, %v, %v", t, v)
		return v
	}
}
