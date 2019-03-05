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

package jsontransform

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
)

// TransformNumbers walks a json decoded tree an replaces json.Number
// with int64, float64, or string, in this order of preference (i.e. if it
// parses as an int, use int. if it parses as a float, use float. etc).
func TransformNumbers(dict common.MapStr) {
	for k, v := range dict {
		switch vv := v.(type) {
		case json.Number:
			dict[k] = transformNumber(vv)
		case map[string]interface{}:
			TransformNumbers(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}

func transformNumber(value json.Number) interface{} {
	i64, err := value.Int64()
	if err == nil {
		return i64
	}
	f64, err := value.Float64()
	if err == nil {
		return f64
	}
	return value.String()
}

func transformNumbersArray(arr []interface{}) {
	for i, v := range arr {
		switch vv := v.(type) {
		case json.Number:
			arr[i] = transformNumber(vv)
		case map[string]interface{}:
			TransformNumbers(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}
