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
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"

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
	err = cp.Copy(filepath.Join("testdata", "container-store"), storePath)
	if err != nil {
		t.Fatalf("cannot copy store files: %s", err)
	}

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
	realStroe := sourceStore{
		identifier: &SourceIdentifier{
			prefix: "filestream" + "::" + "input-id" + "::",
		},
		store: ss,
	}

	return realStroe
}
