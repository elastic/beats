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

package ucfg

import (
	"reflect"
)

// Initializer interface provides initialization of default values support to Unpack.
// The InitDefaults method will be executed for any type passed directly or indirectly to
// Unpack.
type Initializer interface {
	InitDefaults()
}

func tryInitDefaults(val reflect.Value) reflect.Value {
	t := val.Type()

	var initializer Initializer
	if t.Implements(iInitializer) {
		initializer = val.Interface().(Initializer)
		initializer.InitDefaults()
		return val
	} else if reflect.PtrTo(t).Implements(iInitializer) {
		tmp := pointerize(reflect.PtrTo(t), t, val)
		initializer = tmp.Interface().(Initializer)
		initializer.InitDefaults()

		// Return the element in the pointer so the value is set into the
		// field and not a pointer to the value.
		return tmp.Elem()
	}
	return val
}

func hasInitDefaults(t reflect.Type) bool {
	if t.Implements(iInitializer) {
		return true
	}
	if reflect.PtrTo(t).Implements(iInitializer) {
		return true
	}
	return false
}
