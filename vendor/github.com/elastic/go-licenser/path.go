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

package main

import (
	"os"
	"strings"
)

func needsExclusion(path string, exclude []string) bool {
	for _, excluded := range exclude {
		excluded = cleanPathSuffixes(excluded, []string{"*", string(os.PathSeparator)})
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}

	return false
}

func cleanPathSuffixes(path string, sufixes []string) string {
	for _, suffix := range sufixes {
		for strings.HasSuffix(path, suffix) && len(path) > 0 {
			path = path[:len(path)-len(suffix)]
		}
	}

	return path
}

func cleanPathPrefixes(path string, prefixes []string) string {
	for _, prefix := range prefixes {
		for strings.HasPrefix(path, prefix) && len(path) > 0 {
			path = path[len(prefix):]
		}
	}

	return path
}
