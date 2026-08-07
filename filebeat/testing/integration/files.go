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

package integration

import (
	"os"
	"testing"
)

// WriteFile writes content to path, creating the file if needed and truncating any existing content.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// AppendToFile appends content to path, creating the file if it does not exist.
func AppendToFile(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("AppendToFile open(%q): %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("AppendToFile write(%q): %v", path, err)
	}
}

// RotateFile renames src to dst, simulating a log rotation.
func RotateFile(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.Rename(src, dst); err != nil {
		t.Fatalf("RotateFile(%q -> %q): %v", src, dst, err)
	}
}

// TruncateFile truncates path to zero bytes.
func TruncateFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Truncate(path, 0); err != nil {
		t.Fatalf("TruncateFile(%q): %v", path, err)
	}
}

// CreateSymlink creates a symbolic link at linkPath pointing to target.
func CreateSymlink(t *testing.T, target, linkPath string) {
	t.Helper()
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("CreateSymlink(%q -> %q): %v", target, linkPath, err)
	}
}
