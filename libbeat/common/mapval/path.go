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

// PathComponentType indicates the type of PathComponent.
type PathComponentType int

const (
	// PCMapKey is the Type for map keys.
	PCMapKey PathComponentType = 1 + iota
	// PCSliceIdx is the Type for slice indices.
	PCSliceIdx
)

func (pct PathComponentType) String() string {
	if pct == PCMapKey {
		return "map"
	} else if pct == PCSliceIdx {
		return "slice"
	} else {
		// This should never happen, but we don't want to return an
		// error since that would unnecessarily complicate the fluid API
		return "<unknown>"
	}
}

// PathComponent structs represent one breadcrumb in a Path.
type PathComponent struct {
	Type  PathComponentType // One of PCMapKey or PCSliceIdx
	Key   string            // Populated for maps
	Index int               // Populated for slices
}

func (pc PathComponent) String() string {
	if pc.Type == PCSliceIdx {
		return fmt.Sprintf("[%d]", pc.Index)
	}
	return pc.Key
}

// Path represents the path within a nested set of maps.
type Path []PathComponent

// ExtendSlice is used to add a new PathComponent of the PCSliceIdx type.
func (p Path) ExtendSlice(index int) Path {
	return p.extend(
		PathComponent{PCSliceIdx, "", index},
	)
}

// ExtendMap adds a new PathComponent of the PCMapKey type.
func (p Path) ExtendMap(key string) Path {
	return p.extend(
		PathComponent{PCMapKey, key, -1},
	)
}

func (p Path) extend(pc PathComponent) Path {
	out := make(Path, len(p)+1)
	copy(out, p)
	out[len(p)] = pc
	return out
}

// Concat combines two paths into a new path without modifying any existing paths.
func (p Path) Concat(other Path) Path {
	out := make(Path, 0, len(p)+len(other))
	out = append(out, p...)
	return append(out, other...)
}

func (p Path) String() string {
	out := make([]string, len(p))
	for idx, pc := range p {
		out[idx] = pc.String()
	}
	return strings.Join(out, ".")
}

// Last returns a pointer to the last PathComponent in this path. If the path empty,
// a nil pointer is returned.
func (p Path) Last() *PathComponent {
	idx := len(p) - 1
	if idx < 0 {
		return nil
	}
	return &p[len(p)-1]
}

// GetFrom takes a map and fetches the given path from it.
func (p Path) GetFrom(m common.MapStr) (value interface{}, exists bool) {
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
	return fmt.Sprintf("Invalid path Path: %#v", ps)
}

// ParsePath parses a path of form key.[0].otherKey.[1] into a Path object.
func ParsePath(in string) (p Path, err error) {
	keyParts := strings.Split(in, ".")

	p = make(Path, len(keyParts))
	for idx, part := range keyParts {
		r := arrMatcher.FindStringSubmatch(part)
		pc := PathComponent{Index: -1}
		if len(r) > 0 {
			pc.Type = PCSliceIdx
			// Cannot fail, validated by regexp already
			pc.Index, err = strconv.Atoi(r[1])
			if err != nil {
				return p, err
			}
		} else if len(part) > 0 {
			pc.Type = PCMapKey
			pc.Key = part
		} else {
			return p, InvalidPathString(in)
		}

		p[idx] = pc
	}

	return p, nil
}

// MustParsePath is a convenience method for parsing paths that have been previously validated
func MustParsePath(in string) Path {
	out, err := ParsePath(in)
	if err != nil {
		panic(err)
	}
	return out
}
