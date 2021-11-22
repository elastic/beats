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

//go:build windows
// +build windows

package npcap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestNpcap(t *testing.T) {
	const installer = "This is an installer. Honest!\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, installer)
	}))
	defer srv.Close()

	dir, err := os.MkdirTemp("", "packetbeat-npcap-*")
	if err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, CurrentInstaller)

	var hash []byte
	t.Run("Fetch", func(t *testing.T) {
		log := logp.NewLogger("npcap_test_fetch")
		hash, err = Fetch(context.Background(), log, srv.URL, path)
		if err != nil {
			t.Fatalf("failed to fetch installer: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read artifact: %v", err)
		}
		if string(got) != installer {
			t.Errorf("unexpected download: got:%q want:%q", got, installer)
		}
	})

	t.Run("Verify", func(t *testing.T) {
		// Dirty global manipulation. Tests may not be run in parallel.
		hashes["test-artifact"] = "bc3e42210a58873b55554af5100db1f9439b606efde4fd20c98d7af2d6c5419b"
		defer delete(hashes, "test-artifact")

		err = Verify("test-artifact", hash)
		if err != nil {
			t.Errorf("failed to verify download: %v", err)
		}
	})

	t.Run("Install", func(t *testing.T) {
		err := os.Remove(path)
		if err != nil {
			t.Fatalf("failed to remove download: %v", err)
		}
		build := exec.Command("go", "build", "-o", path, filepath.FromSlash("testdata/mock_installer.go"))
		b, err := build.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to build mock installer: %v\n%s", err, b)
		}
		log := logp.NewLogger("npcap_test_install")
		err = Install(context.Background(), log, path, false)
		if err != nil {
			t.Errorf("unexpected error running installer: %v", err)
		}
	})
}
