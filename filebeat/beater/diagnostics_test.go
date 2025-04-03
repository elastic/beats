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

package beater

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestMatchRegistryFiles(t *testing.T) {
	positiveMatches := []string{
		filepath.Join("registry", "filebeat", "49855.json"),
		filepath.Join("registry", "filebeat", "active.dat"),
		filepath.Join("registry", "filebeat", "meta.json"),
		filepath.Join("registry", "filebeat", "log.json"),
	}
	negativeMatches := []string{
		filepath.Join("registry", "filebeat", "bar.dat"),
		filepath.Join("registry", "filebeat", "log.txt"),
		filepath.Join("registry", "42.json"),
		filepath.Join("nop", "active.dat"),
	}
	registryFileRegExps, err := getRegexpsForRegistryFiles()
	if err != nil {
		t.Fatalf("cannot compile regexps for registry paths: %s", err)
	}

	testFn := func(t *testing.T, path string, match bool) {
		result := matchRegistyFiles(registryFileRegExps, path)
		if result != match {
			t.Errorf(
				"mathRegisryFiles('%s') should return %t, got %t instead",
				path,
				match,
				result)
		}
	}

	for _, path := range positiveMatches {
		t.Run(fmt.Sprintf("%s returns true", path), func(t *testing.T) {
			testFn(t, path, true)
		})
	}

	for _, path := range negativeMatches {
		t.Run(fmt.Sprintf("%s returns false", path), func(t *testing.T) {
			testFn(t, path, false)
		})
	}
}
