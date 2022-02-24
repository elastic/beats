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

package safemapstr

import (
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const alternativeKey = "value"

// Put This method implements a way to put dotted keys into a M while
// ensuring they don't override each other. For example:
//
//  a := M{}
//  safemapstr.Put(a, "com.docker.swarm.task", "x")
//  safemapstr.Put(a, "com.docker.swarm.task.id", 1)
//  safemapstr.Put(a, "com.docker.swarm.task.name", "foobar")
//
// Will result in `{"com":{"docker":{"swarm":{"task":{"id":1,"name":"foobar","value":"x"}}}}}`
//
// Put detects this scenario and renames the common base key, by appending
// `.value`
func Put(data mapstr.M, key string, value interface{}) error {
	// XXX This implementation mimics `mapstr.M.Put`, both should be updated to have similar behavior

	d, k := mapFind(data, key, alternativeKey)
	d[k] = value
	return nil
}

// mapFind walk the map based on the given dotted key and returns the final map
// and key to operate on. This function adds intermediate maps, if the key is
// missing from the original map.

// mapFind iterates a M based on the given dotted key, finding the final
// subMap and subKey to operate on.
// If a key is already used, but the used value is no map, an intermediate map will be inserted and
// the old value will be stored using the 'alternativeKey' in a new map.
// If the old value found under key is already an dictionary, subMap will be
// the old value and subKey will be set to alternativeKey.
func mapFind(data mapstr.M, key, alternativeKey string) (subMap mapstr.M, subKey string) {
	// XXX This implementation mimics `common.mapFind`, both should be updated to have similar behavior

	for {
		if oldValue, exists := data[key]; exists {
			if oldMap, ok := tryToM(oldValue); ok {
				return oldMap, alternativeKey
			}
			return data, key
		}

		idx := strings.IndexRune(key, '.')
		if idx < 0 {
			// if old value exists and is a dictionary, return the old dictionary and
			// make sure we store the new value using the 'alternativeKey'
			if oldValue, exists := data[key]; exists {
				if oldMap, ok := tryToM(oldValue); ok {
					return oldMap, alternativeKey
				}
			}

			return data, key
		}

		// Check if first sub-key exists. Create an intermediate map if not.
		k := key[:idx]
		d, exists := data[k]
		if !exists {
			d = mapstr.M{}
			data[k] = d
		}

		// store old value under 'alternativeKey' if the old value is no map.
		// Do not overwrite old value.
		v, ok := tryToM(d)
		if !ok {
			v = mapstr.M{alternativeKey: d}
			data[k] = v
		}

		// advance into sub-map
		key = key[idx+1:]
		data = v
	}
}

func tryToM(v interface{}) (mapstr.M, bool) {
	switch m := v.(type) {
	case mapstr.M:
		return m, true
	case map[string]interface{}:
		return mapstr.M(m), true
	default:
		return nil, false
	}
}
