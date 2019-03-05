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

package mapval

import (
	"reflect"

	"github.com/elastic/beats/libbeat/common"
)

func interfaceToMapStr(o interface{}) common.MapStr {
	newMap := common.MapStr{}
	rv := reflect.ValueOf(o)

	for _, key := range rv.MapKeys() {
		mapV := rv.MapIndex(key)
		keyStr := key.Interface().(string)
		var value interface{}

		if !mapV.IsNil() {
			value = mapV.Interface().(interface{})
		}

		newMap[keyStr] = value
	}
	return newMap
}

func sliceToSliceOfInterfaces(o interface{}) []interface{} {
	rv := reflect.ValueOf(o)
	converted := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		var indexV = rv.Index(i)
		var convertedValue interface{}
		if indexV.Type().Kind() == reflect.Interface {
			if !indexV.IsNil() {
				convertedValue = indexV.Interface().(interface{})
			} else {
				convertedValue = nil
			}
		} else {
			convertedValue = indexV.Interface().(interface{})
		}
		converted[i] = convertedValue
	}
	return converted
}
