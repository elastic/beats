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
	"fmt"
	"reflect"

	"github.com/elastic/go-lookslike/llpath"
)

type walkObserverInfo struct {
	key     llpath.PathComponent
	value   reflect.Value
	rootVal reflect.Value
	path    llpath.Path
}

// walkObserver functions run once per object in the tree.
type walkObserver func(info walkObserverInfo) error

// walk determine if in is a `map[string]interface{}` or a `Slice` and traverse it if so, otherwise will
// treat it as a scalar and invoke the walk observer on the input value directly.
func walk(inVal reflect.Value, expandPaths bool, wo walkObserver) error {
	switch inVal.Kind() {
	case reflect.Map:
		return walkMap(inVal, expandPaths, wo)
	case reflect.Slice:
		return walkSlice(inVal, expandPaths, wo)
	default:
		return walkInterface(inVal, expandPaths, wo)
	}
}

// walkmap[string]interface{} is a shorthand way to walk a tree with a map as the root.
func walkMap(mVal reflect.Value, expandPaths bool, wo walkObserver) error {
	return walkFullMap(mVal, mVal, llpath.Path{}, expandPaths, wo)
}

// walkSlice walks the provided root slice.
func walkSlice(sVal reflect.Value, expandPaths bool, wo walkObserver) error {
	return walkFullSlice(sVal, reflect.ValueOf(map[string]interface{}{}), llpath.Path{}, expandPaths, wo)
}

func walkInterface(s reflect.Value, expandPaths bool, wo walkObserver) error {
	return wo(walkObserverInfo{
		value:   s,
		key:     llpath.PathComponent{},
		rootVal: reflect.ValueOf(map[string]interface{}{}),
		path:    llpath.Path{},
	})
}

func walkFull(oVal, rootVal reflect.Value, path llpath.Path, expandPaths bool, wo walkObserver) (err error) {

	// Unpack any wrapped interfaces
	for oVal.Kind() == reflect.Interface {
		oVal = reflect.ValueOf(oVal.Interface())
	}

	lastPathComponent := path.Last()
	if lastPathComponent == nil {
		// In the case of a slice we can have an empty path
		if oVal.Kind() == reflect.Slice || oVal.Kind() == reflect.Array {
			lastPathComponent = &llpath.PathComponent{}
		} else {
			panic("Attempted to traverse an empty Path on non array/slice in lookslike.walkFull, this should never happen.")
		}
	}

	err = wo(walkObserverInfo{*lastPathComponent, oVal, rootVal, path})
	if err != nil {
		return err
	}

	switch oVal.Kind() {
	case reflect.Map:
		err := walkFullMap(oVal, rootVal, path, expandPaths, wo)
		if err != nil {
			return err
		}
	case reflect.Slice:
		for i := 0; i < oVal.Len(); i++ {
			newPath := path.ExtendSlice(i)
			err := walkFull(oVal.Index(i), rootVal, newPath, expandPaths, wo)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// walkFull walks the given map[string]interface{} tree.
func walkFullMap(mVal, rootVal reflect.Value, p llpath.Path, expandPaths bool, wo walkObserver) (err error) {
	if mVal.Kind() != reflect.Map {
		return fmt.Errorf("could not walk not map type for %s", mVal)
	}

	for _, kVal := range mVal.MapKeys() {
		vVal := mVal.MapIndex(kVal)
		k := kVal.String()

		var newPath llpath.Path
		if !expandPaths {
			newPath = p.ExtendMap(k)
		} else {
			additionalPath, err := llpath.ParsePath(k)
			if err != nil {
				return err
			}
			newPath = p.Concat(additionalPath)
		}

		err = walkFull(vVal, rootVal, newPath, expandPaths, wo)
		if err != nil {
			return err
		}
	}

	return nil
}

func walkFullSlice(sVal reflect.Value, rootVal reflect.Value, p llpath.Path, expandPaths bool, wo walkObserver) (err error) {
	for i := 0; i < sVal.Len(); i++ {
		var newPath llpath.Path
		newPath = p.ExtendSlice(i)

		err = walkFull(sVal.Index(i), rootVal, newPath, expandPaths, wo)
		if err != nil {
			return err
		}
	}

	return nil
}
