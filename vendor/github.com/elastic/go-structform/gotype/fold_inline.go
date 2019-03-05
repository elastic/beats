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

package gotype

import (
	"reflect"

	"github.com/elastic/go-structform"
	"github.com/elastic/go-structform/visitors"
)

// getReflectFoldMapKeys implements inline fold of a map[string]X type,
// not reporting object start/end events
func getReflectFoldMapKeys(c *foldContext, t reflect.Type) (reFoldFn, error) {
	if t.Key().Kind() != reflect.String {
		return nil, errMapRequiresStringKey
	}

	f := getMapInlineByPrimitiveElem(t.Elem())
	if f != nil {
		return f, nil
	}

	elemVisitor, err := getReflectFold(c, t.Elem())
	if err != nil {
		return nil, err
	}

	return makeMapKeysFold(elemVisitor), nil
}

func makeMapKeysFold(elemVisitor reFoldFn) reFoldFn {
	return func(C *foldContext, rv reflect.Value) error {
		if rv.IsNil() || !rv.IsValid() {
			return nil
		}

		for _, k := range rv.MapKeys() {
			if err := C.OnKey(k.String()); err != nil {
				return err
			}
			if err := elemVisitor(C, rv.MapIndex(k)); err != nil {
				return err
			}
		}
		return nil
	}
}

// getReflectFoldInlineInterface create an inline folder for an yet unknown type.
// The actual types folder must open/close an object
func getReflectFoldInlineInterface(C *foldContext, t reflect.Type) (reFoldFn, error) {
	var (
		// cache last used folder
		lastType    reflect.Type
		lastVisitor reFoldFn
	)

	return embeddObjReFold(C, func(C *foldContext, rv reflect.Value) error {
		if rv.Type() != lastType {
			elemVisitor, err := getReflectFold(C, rv.Type())
			if err != nil {
				return err
			}

			lastVisitor = elemVisitor
			lastType = rv.Type()
		}
		return lastVisitor(C, rv)
	}), nil
}

func embeddObjReFold(C *foldContext, objFold reFoldFn) reFoldFn {
	var (
		ctx = *C
		vs  = visitors.NewExpectObjVisitor(nil)
	)

	ctx.visitor = structform.EnsureExtVisitor(vs).(visitor)
	return func(C *foldContext, rv reflect.Value) error {
		// don't inline missing/empty object
		if rv.IsNil() || !rv.IsValid() {
			return nil
		}

		vs.SetActive(C.visitor)
		err := objFold(&ctx, rv)
		if err == nil && !vs.Done() {
			err = errExpectedObjectClose
		}

		vs.SetActive(nil)
		return err
	}
}
