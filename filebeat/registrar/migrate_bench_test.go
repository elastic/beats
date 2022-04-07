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

//go:build linux || darwin
// +build linux darwin

package registrar

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v8/filebeat/input/file"
	libfile "github.com/elastic/beats/v8/libbeat/common/file"
)

var keep bool

func init() {
	flag.BoolVar(&keep, "keep", false, "do not delete test directories")
}

func BenchmarkMigration0To1(b *testing.B) {
	for _, entries := range []int{1, 10, 100, 1000, 10000, 100000} {
		b.Run(fmt.Sprintf("%v", entries), func(b *testing.B) {
			b.StopTimer()

			dataHome := tempDir(b)
			registryHome := filepath.Join(dataHome, "filebeat")
			mkDir(b, registryHome)

			metaPath := filepath.Join(registryHome, "meta.json")
			dataPath := filepath.Join(registryHome, "data.json")

			states := make([]file.State, entries)
			for i := range states {
				states[i] = file.State{
					Id:     fmt.Sprintf("123455-%v", i),
					Source: fmt.Sprintf("/path/to/test/file-%v.log", i),
					FileStateOS: libfile.StateOS{
						Inode:  uint64(i),
						Device: 123455,
					},
				}
			}

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				clearDir(b, registryHome)
				// cleanup older run

				writeFile(b, metaPath, []byte(`{"version": "0"}`))
				func() {
					f, err := os.Create(dataPath)
					if err != nil {
						b.Fatal(err)
					}
					defer f.Close()

					enc := json.NewEncoder(f)
					if err := enc.Encode(states); err != nil {
						b.Fatal(err)
					}
				}()

				migrator := &Migrator{
					dataPath:    dataHome,
					permissions: 0o600,
				}

				b.StartTimer()
				if err := migrator.updateToVersion1(registryHome); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func tempDir(t testing.TB) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	path, err := ioutil.TempDir(cwd, "")
	if err != nil {
		t.Fatal(err)
	}

	if !keep {
		t.Cleanup(func() {
			os.RemoveAll(path)
		})
	}
	return path
}

func mkDir(t testing.TB, path string) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func clearDir(t testing.TB, path string) {
	old, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range old {
		if err := os.RemoveAll(info.Name()); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t testing.TB, path string, contents []byte) {
	t.Helper()
	err := ioutil.WriteFile(path, contents, 0o600)
	if err != nil {
		t.Fatal(err)
	}
}
