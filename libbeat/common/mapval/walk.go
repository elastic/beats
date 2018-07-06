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

type walkObserverInfo struct {
	key     PathComponent
	value   interface{}
	rootMap common.MapStr
	path    Path
}

// walkObserver functions run once per object in the tree.
type walkObserver func(info walkObserverInfo) error

// walk is a shorthand way to walk a tree.
func walk(m common.MapStr, expandPaths bool, wo walkObserver) error {
	return walkFullMap(m, m, Path{}, expandPaths, wo)
}

func walkFull(o interface{}, root common.MapStr, path Path, expandPaths bool, wo walkObserver) (err error) {
	lastPathComponent := path.Last()
	if lastPathComponent == nil {
		panic("Attempted to traverse an empty path in mapval.walkFull, this should never happen.")
	}

	err = wo(walkObserverInfo{*lastPathComponent, o, root, path})
	if err != nil {
		return err
	}

	switch reflect.TypeOf(o).Kind() {
	case reflect.Map:
		converted := interfaceToMapStr(o)
		err := walkFullMap(converted, root, path, expandPaths, wo)
		if err != nil {
			return err
		}
	case reflect.Slice:
		converted := sliceToSliceOfInterfaces(o)

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

// walkFullMap walks the given MapStr tree.
func walkFullMap(m common.MapStr, root common.MapStr, path Path, expandPaths bool, wo walkObserver) (err error) {
	for k, v := range m {
		var newPath Path
		if !expandPaths {
			newPath = path.ExtendMap(k)
		} else {
			additionalPath, err := ParsePath(k)
			if err != nil {
				return err
			}
			newPath = path.Concat(additionalPath)
		}

		err = walkFull(v, root, newPath, expandPaths, wo)
		if err != nil {
			return err
		}
	}

	return nil
}
