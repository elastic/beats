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

package cptest

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func makeKey(s string) []byte {
	return []byte(s)
}

func makeKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = makeKey(fmt.Sprintf("key%v", i))
	}
	return keys
}

func withBackend(factory BackendFactory, fn func(*testing.T, BackendFactory)) func(*testing.T) {
	return func(t *testing.T) {
		fn(t, factory)
	}
}

func runWithBools(t *testing.T, name string, fn func(*testing.T, bool)) {
	withBools(name, fn)(t)
}

func withBools(name string, fn func(*testing.T, bool)) func(t *testing.T) {
	return func(t *testing.T) {
		for _, b := range []bool{false, true} {
			b := b
			t.Run(fmt.Sprintf("%v=%v", name, b), func(t *testing.T) {
				fn(t, b)
			})
		}
	}
}

func RunWithPath(t *testing.T, factory BackendFactory, fn func(*Registry)) {
	WithPath(factory, func(_ *testing.T, reg *Registry) {
		fn(reg)
	})(t)
}

func WithPath(factory BackendFactory, fn func(*testing.T, *Registry)) func(t *testing.T) {
	return func(t *testing.T) {
		path, err := ioutil.TempDir(defaultTempDir, "")
		if err != nil {
			t.Fatalf("Failed to create temporary test directory: %v", err)
		}

		t.Logf("Test tmp dir: %v", path)
		if !keepTmpDir {
			defer os.RemoveAll(path)
		}

		reg, err := factory(path)
		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}
		defer reg.Close()

		fn(t, &Registry{T: t, Registry: reg})
	}
}

func RunWithStore(t *testing.T, factory BackendFactory, fn func(*Store)) {
	WithStore(factory, func(_ *testing.T, store *Store) {
		fn(store)
	})(t)
}

func WithStore(factory BackendFactory, fn func(*testing.T, *Store)) func(*testing.T) {
	return WithPath(factory, func(t *testing.T, reg *Registry) {
		store := reg.Access("test")
		defer store.Close()
		fn(t, store)
	})
}

func must(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatal(msg, ":", err)
	}
}
