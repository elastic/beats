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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// expandFields de-dots the keys in m by expanding them in-place into a
// nested object structure, merging objects as necessary. If there are any
// conflicts (i.e. a common prefix where one field is an object and another
// is a non-object), an error will be returned.
//
// Note that expandFields is destructive, and in the case of an error the
// map may be left in a semi-expanded state.
func expandFields(m mapstr.M) error {
	for k, v := range m {
		newMap, newIsMap := getMap(v)
		if newIsMap {
			if err := expandFields(newMap); err != nil {
				return errors.Wrapf(err, "error expanding %q", k)
			}
		}
		if dot := strings.IndexRune(k, '.'); dot < 0 {
			continue
		}

		// Delete the dotted key.
		delete(m, k)

		// Put expands k, returning the original value if any.
		//
		// If v is a map then we will merge with an existing map if any,
		// otherwise there must not be an existing value.
		old, err := m.Put(k, v)
		if err != nil {
			// Put will return an error if we attempt to insert into a non-object value.
			return fmt.Errorf("cannot expand %q: found conflicting key", k)
		}
		if old == nil {
			continue
		}
		if !newIsMap {
			return fmt.Errorf("cannot expand %q: found existing (%T) value", k, old)
		} else {
			oldMap, oldIsMap := getMap(old)
			if !oldIsMap {
				return fmt.Errorf("cannot expand %q: found conflicting key", k)
			}
			if err := mergeObjects(newMap, oldMap); err != nil {
				return errors.Wrapf(err, "cannot expand %q", k)
			}
		}
	}
	return nil
}

// mergeObjects deep merges the elements of rhs into lhs.
//
// mergeObjects will recursively combine the entries of
// objects with the same key in each object. If there exist
// two entries with the same key in each object which
// are not both objects, then an error will result.
func mergeObjects(lhs, rhs mapstr.M) error {
	for k, rhsValue := range rhs {
		lhsValue, ok := lhs[k]
		if !ok {
			lhs[k] = rhsValue
			continue
		}
		lhsMap, ok := getMap(lhsValue)
		if !ok {
			return fmt.Errorf("cannot merge %q: found (%T) value", k, lhsValue)
		}
		rhsMap, ok := getMap(rhsValue)
		if !ok {
			return fmt.Errorf("cannot merge %q: found (%T) value", k, rhsValue)
		}
		if err := mergeObjects(lhsMap, rhsMap); err != nil {
			return errors.Wrapf(err, "cannot merge %q", k)
		}
	}
	return nil
}

func getMap(v interface{}) (map[string]interface{}, bool) {
	switch v := v.(type) {
	case map[string]interface{}:
		return v, true
	case mapstr.M:
		return v, true
	}
	return nil, false
}
