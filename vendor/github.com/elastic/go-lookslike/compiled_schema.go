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

package lookslike

import (
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
	"reflect"
)

type flatValidator struct {
	path  llpath.Path
	isDef isdef.IsDef
}

// CompiledSchema represents a compiled definition for driving a validator.Validator.
type CompiledSchema []flatValidator

// Check executes the the checks within the CompiledSchema
func (cs CompiledSchema) Check(actual interface{}) *llresult.Results {
	res := llresult.NewResults()
	for _, pv := range cs {
		actualVal, actualKeyExists := pv.path.GetFrom(reflect.ValueOf(actual))
		var actualInter interface{}
		zero := reflect.Value{}
		if actualVal != zero {
			actualInter = actualVal.Interface()
		}

		if !pv.isDef.Optional || pv.isDef.Optional && actualKeyExists {
			var checkRes *llresult.Results
			checkRes = pv.isDef.Check(pv.path, actualInter, actualKeyExists)
			res.Merge(checkRes)
		}
	}

	return res
}
