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

	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

// RunWithPath uses the factory to create and configure a registry with a
// temporary test path. The test function fn is called with the new registry.
// The registry is closed once the test finishes and the temporary is deleted
// afterwards (unless the `-keep` CLI flag is used).
func RunWithPath(t *testing.T, factory BackendFactory, fn func(*Registry)) {
	WithPath(factory, func(_ *testing.T, reg *Registry) {
		fn(reg)
	})(t)
}

// WithPath wraps a registry aware test function into a normalized test
// function that can be used with `t.Run`.
// The factory is used to create and configure the registry with a temporary
// test path.  The registry is closed and the temporary test directoy is delete
// if the test function returns or panics.
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

// RunWithStore uses the factory to create a registry and temporary store, that
// is used with fn.  The temporary directory used for the store is deleted once
// fn returns.
func RunWithStore(t *testing.T, factory BackendFactory, fn func(*Store)) {
	WithStore(factory, func(_ *testing.T, store *Store) {
		fn(store)
	})(t)
}

// WithStore wraps a store aware test function into a normalized test function
// that can be used with `t.Run`.  WithStore is based on WithPath, but will
// create and pass a test store (named "test") to the test function. The test
// store is closed once the test function returns or panics.
func WithStore(factory BackendFactory, fn func(*testing.T, *Store)) func(*testing.T) {
	return WithPath(factory, func(t *testing.T, reg *Registry) {
		store := reg.Access("test")
		defer store.Close()
		fn(t, store)
	})
}

func makeKey(s string) backend.Key {
	return backend.Key(s)
}

func makeKeys(n int) []backend.Key {
	keys := make([]backend.Key, n)
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

func must(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatal(msg, ":", err)
	}
}
