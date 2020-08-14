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

package conditions

import "fmt"

// Fields represents an arbitrary map in a config file.
type Fields struct {
	fields map[string]interface{}
}

// Unpack unpacks nested fields set with dot notation like foo.bar into the proper nesting
// in a nested map/slice structure.
func (f *Fields) Unpack(to interface{}) error {
	m, ok := to.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type, expect map")
	}

	f.fields = map[string]interface{}{}

	var expand func(key string, value interface{})

	expand = func(key string, value interface{}) {
		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range v {
				expand(fmt.Sprintf("%v.%v", key, k), val)
			}
		case []interface{}:
			for i := range v {
				expand(fmt.Sprintf("%v.%v", key, i), v[i])
			}
		default:
			f.fields[key] = value
		}
	}

	for k, val := range m {
		expand(k, val)
	}
	return nil
}
