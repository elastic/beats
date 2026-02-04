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

//go:build integration

package input_logfile

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestTakeOverCanReadLogInputStatesWithMeta(t *testing.T) {
	rootDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "build", "integration-tests"))
	if err != nil {
		t.Fatalf("cannot get abs path: %s", err)
	}
	tmpDir := integration.CreateTempDir(
		t,
		rootDir,
	)
	name := "filebeat"
	storePath := filepath.Join(tmpDir, name)

	// Copy a real example for testing
	copyFolder(t, filepath.Join("testdata", "container-store"), storePath)
	store := openTestStoreFromFiles(t, tmpDir, name)

	count := 0
	store.TakeOver(func(v TakeOverState) (string, any) {
		// Count the states successfully read
		count++
		return "", nil
	})

	// Ensure we read all states
	if count != 2 {
		t.Fatalf("could not read/convert Log input states, look the error logs at %s", tmpDir)
	}
}

func openTestStoreFromFiles(t *testing.T, rootDir, name string) sourceStore {
	logger := logptest.NewFileLogger(t, rootDir)
	reg, err := memlog.New(logger.Logger, memlog.Settings{
		Root: rootDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	storeReg := statestore.NewRegistry(reg)
	store, err := storeReg.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	tstore := testStateStore{
		Store: store,
	}

	ss, err := openStore(logger.Logger, tstore, "filestream")
	if err != nil {
		t.Fatalf("cannot open store: %s", err)
	}
	realStore := sourceStore{
		identifier: &SourceIdentifier{
			prefix: "filestream" + "::" + "input-id" + "::",
		},
		store: ss,
	}

	return realStore
}

func copy(t *testing.T, src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("cannot open source file: %s", err)
	}
	defer srcFile.Close()

	st, err := srcFile.Stat()
	if err != nil {
		t.Fatalf("cannot stat source file: %s", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		t.Fatalf("cannot open dst file: %s", err)
	}
	defer func() {
		if err := dstFile.Sync(); err != nil {
			t.Fatalf("cannot sync dst file: %s", err)
		}
		if err := dstFile.Close(); err != nil {
			t.Fatalf("cannot close dst file: %s", err)
		}
	}()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("cannot copy file: %s", err)
	}

	return nil
}

func copyFolder(t *testing.T, src, dst string) {
	if err := os.MkdirAll(dst, 0o777); err != nil {
		t.Fatalf("cannot create dst folder: %s", err)
	}

	filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		// Only copy files
		if d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		copy(t, path, filepath.Join(dst, base))
		return nil
	})
}
