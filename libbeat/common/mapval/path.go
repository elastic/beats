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
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// pathComponentType indicates the type of pathComponent.
type pathComponentType int

const (
	// pcMapKey is the Type for map keys.
	pcMapKey pathComponentType = 1 + iota
	// pcSliceIdx is the Type for slice indices.
	pcSliceIdx
)

func (pct pathComponentType) String() string {
	if pct == pcMapKey {
		return "map"
	} else if pct == pcSliceIdx {
		return "slice"
	} else {
		// This should never happen, but we don't want to return an
		// error since that would unnecessarily complicate the fluid API
		return "<unknown>"
	}
}

// pathComponent structs represent one breadcrumb in a path.
type pathComponent struct {
	Type  pathComponentType // One of pcMapKey or pcSliceIdx
	Key   string            // Populated for maps
	Index int               // Populated for slices
}

func (pc pathComponent) String() string {
	if pc.Type == pcSliceIdx {
		return fmt.Sprintf("[%d]", pc.Index)
	}
	return pc.Key
}

// path represents the path within a nested set of maps.
type path []pathComponent

// extendSlice is used to add a new pathComponent of the pcSliceIdx type.
func (p path) extendSlice(index int) path {
	return p.extend(
		pathComponent{pcSliceIdx, "", index},
	)
}

// extendMap adds a new pathComponent of the pcMapKey type.
func (p path) extendMap(key string) path {
	return p.extend(
		pathComponent{pcMapKey, key, -1},
	)
}

func (p path) extend(pc pathComponent) path {
	out := make(path, len(p)+1)
	copy(out, p)
	out[len(p)] = pc
	return out
}

// concat combines two paths into a new path without modifying any existing paths.
func (p path) concat(other path) path {
	out := make(path, 0, len(p)+len(other))
	out = append(out, p...)
	return append(out, other...)
}

func (p path) String() string {
	out := make([]string, len(p))
	for idx, pc := range p {
		out[idx] = pc.String()
	}
	return strings.Join(out, ".")
}

// last returns a pointer to the last pathComponent in this path. If the path empty,
// a nil pointer is returned.
func (p path) last() *pathComponent {
	idx := len(p) - 1
	if idx < 0 {
		return nil
	}
	return &p[len(p)-1]
}

// getFrom takes a map and fetches the given path from it.
func (p path) getFrom(m common.MapStr) (value interface{}, exists bool) {
	value = m
	exists = true
	for _, pc := range p {
		rt := reflect.TypeOf(value)
		switch rt.Kind() {
		case reflect.Map:
			converted := interfaceToMapStr(value)
			value, exists = converted[pc.Key]
		case reflect.Slice:
			converted := sliceToSliceOfInterfaces(value)
			if pc.Index < len(converted) {
				exists = true
				value = converted[pc.Index]
			} else {
				exists = false
				value = nil
			}
		default:
			// If this case has been reached this means the expected type, say a map,
			// is actually something else, like a string or an array. In this case we
			// simply say the value doesn't exist. From a practical perspective this is
			// the right behavior since it will cause validation to fail.
			return nil, false
		}

		if exists == false {
			return nil, exists
		}
	}

	return value, exists
}

var arrMatcher = regexp.MustCompile("\\[(\\d+)\\]")

// InvalidPathString is the error type returned from unparseable paths.
type InvalidPathString string

func (ps InvalidPathString) Error() string {
	return fmt.Sprintf("Invalid path path: %#v", ps)
}

// parsePath parses a path of form key.[0].otherKey.[1] into a path object.
func parsePath(in string) (p path, err error) {
	keyParts := strings.Split(in, ".")

	p = make(path, len(keyParts))
	for idx, part := range keyParts {
		r := arrMatcher.FindStringSubmatch(part)
		pc := pathComponent{Index: -1}
		if len(r) > 0 {
			pc.Type = pcSliceIdx
			// Cannot fail, validated by regexp already
			pc.Index, err = strconv.Atoi(r[1])
			if err != nil {
				return p, err
			}
		} else if len(part) > 0 {
			pc.Type = pcMapKey
			pc.Key = part
		} else {
			return p, InvalidPathString(in)
		}

		p[idx] = pc
	}

	return p, nil
}

// mustParsePath is a convenience method for parsing paths that have been previously validated
func mustParsePath(in string) path {
	out, err := parsePath(in)
	if err != nil {
		panic(err)
	}
	return out
}
