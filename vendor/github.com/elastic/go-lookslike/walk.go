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
	"reflect"

	"github.com/elastic/go-lookslike/internal/llreflect"
	"github.com/elastic/go-lookslike/llpath"
)

type walkObserverInfo struct {
	key   llpath.PathComponent
	value interface{}
	root  map[string]interface{}
	path  llpath.Path
}

// walkObserver functions run once per object in the tree.
type walkObserver func(info walkObserverInfo) error

// walk determine if in is a `map[string]interface{}` or a `Slice` and traverse it if so, otherwise will
// treat it as a scalar and invoke the walk observer on the input value directly.
func walk(in interface{}, expandPaths bool, wo walkObserver) error {
	switch in.(type) {
	case map[string]interface{}:
		return walkMap(in.(map[string]interface{}), expandPaths, wo)
	case []interface{}:
		return walkSlice(in.([]interface{}), expandPaths, wo)
	default:
		return walkInterface(in, expandPaths, wo)
	}
}

// walkmap[string]interface{} is a shorthand way to walk a tree with a map as the root.
func walkMap(m map[string]interface{}, expandPaths bool, wo walkObserver) error {
	return walkFullMap(m, m, llpath.Path{}, expandPaths, wo)
}

// walkSlice walks the provided root slice.
func walkSlice(s []interface{}, expandPaths bool, wo walkObserver) error {
	return walkFullSlice(s, map[string]interface{}{}, llpath.Path{}, expandPaths, wo)
}

func walkInterface(s interface{}, expandPaths bool, wo walkObserver) error {
	return wo(walkObserverInfo{
		value: s,
		key:   llpath.PathComponent{},
		root:  map[string]interface{}{},
		path:  llpath.Path{},
	})
}

func walkFull(o interface{}, root map[string]interface{}, path llpath.Path, expandPaths bool, wo walkObserver) (err error) {
	lastPathComponent := path.Last()
	if lastPathComponent == nil {
		// In the case of a slice we can have an empty path
		if _, ok := o.([]interface{}); ok {
			lastPathComponent = &llpath.PathComponent{}
		} else {
			panic("Attempted to traverse an empty Path on a map[string]interface{} in lookslike.walkFull, this should never happen.")
		}
	}

	err = wo(walkObserverInfo{*lastPathComponent, o, root, path})
	if err != nil {
		return err
	}

	switch reflect.TypeOf(o).Kind() {
	case reflect.Map:
		converted := llreflect.InterfaceToMap(o)
		err := walkFullMap(converted, root, path, expandPaths, wo)
		if err != nil {
			return err
		}
	case reflect.Slice:
		converted := llreflect.InterfaceToSliceOfInterfaces(o)

		for idx, v := range converted {
			newPath := path.ExtendSlice(idx)
			err := walkFull(v, root, newPath, expandPaths, wo)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// walkFull walks the given map[string]interface{} tree.
func walkFullMap(m map[string]interface{}, root map[string]interface{}, p llpath.Path, expandPaths bool, wo walkObserver) (err error) {
	for k, v := range m {
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

		err = walkFull(v, root, newPath, expandPaths, wo)
		if err != nil {
			return err
		}
	}

	return nil
}

func walkFullSlice(s []interface{}, root map[string]interface{}, p llpath.Path, expandPaths bool, wo walkObserver) (err error) {
	for idx, v := range s {
		var newPath llpath.Path
		newPath = p.ExtendSlice(idx)

		err = walkFull(v, root, newPath, expandPaths, wo)
		if err != nil {
			return err
		}
	}

	return nil
}
