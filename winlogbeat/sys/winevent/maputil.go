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

package winevent

import (
	"fmt"
	"reflect"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/winlogbeat/sys"
)

// AddOptional adds a key and value to the given MapStr if the value is not the
// zero value for the type of v. It is safe to call the function with a nil
// MapStr.
func AddOptional(m common.MapStr, key string, v interface{}) {
	if m != nil && !isZero(v) {
		_, _ = m.Put(key, v)
	}
}

// AddPairs adds a new dictionary to the given MapStr. The key/value pairs are
// added to the new dictionary. If any keys are duplicates, the first key/value
// pair is added and the remaining duplicates are dropped.
//
// The new dictionary is added to the given MapStr and it is also returned for
// convenience purposes.
func AddPairs(m common.MapStr, key string, pairs []KeyValue) common.MapStr {
	if len(pairs) == 0 {
		return nil
	}

	h := make(common.MapStr, len(pairs))
	for i, kv := range pairs {
		// Ignore empty values.
		if kv.Value == "" {
			continue
		}

		// If the key name is empty or if it the default of "Data" then
		// assign a generic name of paramN.
		k := kv.Key
		if k == "" || k == "Data" {
			k = fmt.Sprintf("param%d", i+1)
		}

		// Do not overwrite.
		_, err := h.GetValue(k)
		if err == common.ErrKeyNotFound {
			_, _ = h.Put(k, sys.RemoveWindowsLineEndings(kv.Value))
		} else {
			debugf("Dropping key/value (k=%s, v=%s) pair because key already "+
				"exists. event=%+v", k, kv.Value, m)
		}
	}

	if len(h) == 0 {
		return nil
	}

	_, _ = m.Put(key, h)

	return h
}

// isZero return true if the given value is the zero value for its type.
func isZero(i interface{}) bool {
	switch i := i.(type) {
	case nil:
		return true
	case time.Time:
		return false
	default:
		return reflect.ValueOf(i).IsZero()
	}
}
