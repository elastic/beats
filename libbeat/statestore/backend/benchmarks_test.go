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

package backend_test

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/otelstorage"
	"github.com/elastic/elastic-agent-libs/logp"
)

const storeName = "benchstore"

type registryFactory struct {
	name    string
	newFunc func(dir string) (backend.Registry, error)
}

var factories = []registryFactory{
	{
		name: "memlog",
		newFunc: func(dir string) (backend.Registry, error) {
			return memlog.New(logp.NewNopLogger(), memlog.Settings{
				Root:     dir,
				FileMode: 0o600,
			})
		},
	},
	{
		name: "otel_file_storage",
		newFunc: func(dir string) (backend.Registry, error) {
			cfg := otelstorage.DefaultFileStorageConfig()
			cfg.Directory = dir
			cfg.CreateDirectory = true
			return otelstorage.NewFileStorage(otelstorage.Settings{
				Config:     cfg,
				ReceiverID: component.MustNewID("bench"),
				Logger:     logp.NewNopLogger(),
			})
		},
	},
}

func makeValue(i int) map[string]any {
	return map[string]any{
		"cursor": map[string]any{
			"offset": fmt.Sprintf("%d", i),
		},
		"source": fmt.Sprintf("/var/log/app-%d.log", i),
		"ttl":    "300s",
	}
}

func key(i int) string {
	return fmt.Sprintf("filestream::input-%d", i)
}

func BenchmarkCRUD(b *testing.B) {
	const numKeys = 10000

	for _, f := range factories {
		b.Run(f.name, func(b *testing.B) {
			dir := b.TempDir()
			reg, err := f.newFunc(dir)
			if err != nil {
				b.Fatal(err)
			}
			defer reg.Close()

			store, err := reg.Access(storeName)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()

			b.Run("Set", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if err := store.Set(key(i%numKeys), makeValue(i)); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Get", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var got map[string]any
					if err := store.Get(key(i%numKeys), &got); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Has", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if _, err := store.Has(key(i % numKeys)); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Remove", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					k := key(i % numKeys)
					_ = store.Set(k, makeValue(i))
					if err := store.Remove(k); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("Each", func(b *testing.B) {
				for i := 0; i < numKeys; i++ {
					if err := store.Set(key(i), makeValue(i)); err != nil {
						b.Fatal(err)
					}
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = store.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
						return true, nil
					})
				}
			})

			reportHeap(b, f.name)
			reportDirSize(b, f.name, dir)
		})
	}
}

// BenchmarkFilestreamHotpath simulates ingestion of 10K files as filestream
// does it. For each file the benchmark creates a registry entry and then
// updates the cursor offset 1000 times (simulating line-by-line ACKs).
func BenchmarkFilestreamHotpath(b *testing.B) {
	const (
		numFiles             = 10_000
		cursorUpdatesPerFile = 1000
	)

	for _, f := range factories {
		b.Run(f.name, func(b *testing.B) {
			dir := b.TempDir()
			reg, err := f.newFunc(dir)
			if err != nil {
				b.Fatal(err)
			}
			defer reg.Close()

			store, err := reg.Access(storeName)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()

			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				for i := 0; i < numFiles; i++ {
					k := fmt.Sprintf("filestream::native_id-%d", i)
					src := fmt.Sprintf("/var/log/service/app-%d.log", i)
					for u := 0; u <= cursorUpdatesPerFile; u++ {
						state := map[string]any{
							"ttl":     "1800s",
							"updated": "2026-04-17T10:00:00Z",
							"cursor":  map[string]any{"offset": fmt.Sprintf("%d", u*512), "eof": u == cursorUpdatesPerFile},
							"meta":    map[string]any{"source": src, "identifier_name": "native"},
						}
						if err := store.Set(k, state); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
			b.StopTimer()
			reportHeap(b, f.name)
			reportDirSize(b, f.name, dir)
		})
	}
}

func reportDirSize(b *testing.B, backendName, dir string) {
	b.Helper()
	var totalBytes int64
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		files = append(files, fmt.Sprintf("  %s  %d bytes", rel, info.Size()))
		totalBytes += info.Size()
		return nil
	})
	b.Logf("\n[%s] data directory: %s", backendName, dir)
	for _, line := range files {
		b.Logf("%s", line)
	}
	b.Logf("  TOTAL: %d bytes (%.2f KB)", totalBytes, float64(totalBytes)/1024)
}

func reportHeap(b *testing.B, backendName string) {
	b.Helper()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.Logf("[%s] heap: Alloc=%d bytes (%.2f KB), HeapInuse=%d bytes (%.2f KB), HeapObjects=%d",
		backendName, m.Alloc, float64(m.Alloc)/1024, m.HeapInuse, float64(m.HeapInuse)/1024, m.HeapObjects)
}
