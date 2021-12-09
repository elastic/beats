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

package npcap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestNpcap(t *testing.T) {
	// Ugh.
	var lcfg logp.Config
	logp.ToObserverOutput()(&lcfg)
	logp.Configure(lcfg)
	obs := logp.ObserverLogs()

	// Working space.
	dir, err := os.MkdirTemp("", "packetbeat-npcap-*")
	if err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "installer")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}

	t.Run("Install", func(t *testing.T) {
		build := exec.Command("go", "build", "-o", path, filepath.FromSlash("testdata/mock_installer.go"))
		b, err := build.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to build mock installer: %v\n%s", err, b)
		}
		log := logp.NewLogger("npcap_test_install")
		for _, compat := range []bool{false, true} {
			for _, dst := range []string{
				"", // Default.
				`C:\some\other\location`,
			} {
				err = install(context.Background(), log, path, dst, compat)
				messages := obs.TakeAll()
				if err != nil {
					if dst == "" {
						dst = "default location"
					}
					t.Errorf("unexpected error running installer to %s with compat=%t: %v", dst, compat, err)
					for _, e := range messages {
						t.Log(e.Message)
					}
				}
			}
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		path = filepath.Join(filepath.Dir(path), "Uninstall.exe")
		build := exec.Command("go", "build", "-o", path, filepath.FromSlash("testdata/mock_uninstaller.go"))
		b, err := build.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to build mock uninstaller: %v\n%s", err, b)
		}
		log := logp.NewLogger("npcap_test_uninstall")
		err = uninstall(context.Background(), log, path)
		messages := obs.TakeAll()
		if err != nil {
			t.Errorf("unexpected error running uninstaller: %v", err)
			for _, e := range messages {
				t.Log(e.Message)
			}
		}
	})
}
