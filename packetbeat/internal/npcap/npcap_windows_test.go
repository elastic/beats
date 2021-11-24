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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestNpcap(t *testing.T) {
	// Elements of truth.
	const (
		registryEndPoint    = "/npcap/latest_installer"
		latestVersion       = "0.0"
		latestInstaller     = "npcap-0.0-oem.exe"
		latestHash          = "bc3e42210a58873b55554af5100db1f9439b606efde4fd20c98d7af2d6c5419b"
		latestInstallerPath = "/npcap/" + latestInstaller
		installer           = "This is an installer. Honest!\n"
	)
	var latestVersionInfo string

	// Mock registry and download server.
	mux := http.NewServeMux()
	mux.HandleFunc(registryEndPoint, func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, latestVersionInfo)
	})
	mux.HandleFunc(latestInstallerPath, func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, installer)
	})
	srv := httptest.NewServer(mux)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse server root URL: %v", err)
	}
	u.Path = latestInstallerPath
	latestInstallerURL := u.String()
	latestVersionInfo = `{"url":"` + latestInstallerURL + `","version":"` + latestVersion + `","hash":"` + latestHash + `"}`
	defer srv.Close()

	// Working space.
	dir, err := os.MkdirTemp("", "packetbeat-npcap-*")
	if err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, latestVersion)

	t.Run("Query Version", func(t *testing.T) {
		u.Path = registryEndPoint

		log := logp.NewLogger("npcap_test_query_version")
		gotVersion, gotURL, gotHash, err := CurrentVersion(context.Background(), log, u.String())
		if err != nil {
			t.Fatalf("failed to fetch installer: %v", err)
		}

		if gotVersion != latestVersion {
			t.Errorf("unexpected version: got:%q want:%q", gotVersion, latestVersion)
		}
		if gotURL != latestInstallerURL {
			t.Errorf("unexpected download location: got:%q want:%q", gotURL, latestInstallerURL)
		}
		if gotHash != latestHash {
			t.Errorf("unexpected hash: got:%q want:%q", gotHash, latestHash)
		}
	})

	t.Run("Fetch", func(t *testing.T) {
		log := logp.NewLogger("npcap_test_fetch")
		hash, err := Fetch(context.Background(), log, latestInstallerURL, path)
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
		if hash != latestHash {
			t.Errorf("unexpected download hash: got:%s want:%s", hash, latestHash)
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
