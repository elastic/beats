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
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type walkObserverInfo struct {
	key        string
	value      interface{}
	currentMap common.MapStr
	rootMap    common.MapStr
	path       []string
	dottedPath string
}

// walkObserver functions run once per object in the tree.
type walkObserver func(info walkObserverInfo)

// walk is a shorthand way to walk a tree.
func walk(m common.MapStr, wo walkObserver) {
	walkFull(m, m, []string{}, wo)
}

// walkFull walks the given MapStr tree.
// TODO: Handle slices/arrays. We intentionally don't handle list types now because we don't need it (yet)
// and it isn't clear in the context of validation what the right thing is to do there beyond letting the user
// perform a custom validation
func walkFull(m common.MapStr, root common.MapStr, path []string, wo walkObserver) {
	for k, v := range m {
		splitK := strings.Split(k, ".")
		newPath := make([]string, len(path)+len(splitK))
		copy(newPath, path)
		copy(newPath[len(path):], splitK)

		dottedPath := strings.Join(newPath, ".")

		wo(walkObserverInfo{k, v, m, root, newPath, dottedPath})

		// Walk nested maps
		vIsMap := false
		var mapV common.MapStr
		if convertedMS, ok := v.(common.MapStr); ok {
			mapV = convertedMS
			vIsMap = true
		} else if convertedM, ok := v.(Map); ok {
			mapV = common.MapStr(convertedM)
			vIsMap = true
		}

		if vIsMap {
			walkFull(mapV, root, newPath, wo)
		}
	}
}
