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

package mapstr

import (
	structform "github.com/elastic/go-structform"
	"github.com/elastic/go-structform/gotype"
)

func foldSlice(v structform.ExtVisitor, s []interface{}) error {
	if err := v.OnArrayStart(len(s), structform.AnyType); err != nil {
		return err
	}
	for _, val := range s {
		switch x := val.(type) {
		case string:
			if err := v.OnString(x); err != nil {
				return err
			}
		case int:
			if err := v.OnInt(x); err != nil {
				return err
			}
		case int64:
			if err := v.OnInt64(x); err != nil {
				return err
			}
		case float64:
			if err := v.OnFloat64(x); err != nil {
				return err
			}
		case bool:
			if err := v.OnBool(x); err != nil {
				return err
			}
		case nil:
			if err := v.OnNil(); err != nil {
				return err
			}
		case M:
			if err := x.Fold(v); err != nil {
				return err
			}
		case map[string]interface{}:
			if err := M(x).Fold(v); err != nil {
				return err
			}
		default:
			if err := gotype.Fold(val, v); err != nil {
				return err
			}
		}
	}
	return v.OnArrayFinished()
}

// Fold implements go-structform's gotype.Folder interface, letting
// go-structform serialize M without the reflect.Convert overhead it
// normally incurs for named map types.
//
// The type switch handles common leaf types (string, int, int64, float64,
// bool, nil) directly rather than delegating to gotype.Fold. This is
// intentional: each gotype.Fold call allocates an iterator and type
// registry (~28 B), so falling back for every leaf value adds 20-72
// allocs/op depending on event shape (2x slower on typical events).
// Uncommon types still fall through to gotype.Fold for full coverage.
func (m M) Fold(v structform.ExtVisitor) error {
	if err := v.OnObjectStart(len(m), structform.AnyType); err != nil {
		return err
	}
	for k, val := range m {
		if err := v.OnKey(k); err != nil {
			return err
		}
		switch x := val.(type) {
		case string:
			if err := v.OnString(x); err != nil {
				return err
			}
		case int:
			if err := v.OnInt(x); err != nil {
				return err
			}
		case int64:
			if err := v.OnInt64(x); err != nil {
				return err
			}
		case float64:
			if err := v.OnFloat64(x); err != nil {
				return err
			}
		case bool:
			if err := v.OnBool(x); err != nil {
				return err
			}
		case nil:
			if err := v.OnNil(); err != nil {
				return err
			}
		case M:
			if err := x.Fold(v); err != nil {
				return err
			}
		case map[string]interface{}:
			if err := M(x).Fold(v); err != nil {
				return err
			}
		case []interface{}:
			if err := foldSlice(v, x); err != nil {
				return err
			}
		default:
			if err := gotype.Fold(val, v); err != nil {
				return err
			}
		}
	}
	return v.OnObjectFinished()
}
