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

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || windows

//nolint:errorlint // Bad linter!
package route

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/execabs"
)

func TestDefault(t *testing.T) {
	for _, family := range []int{syscall.AF_INET, syscall.AF_INET6} {
		wantIface, wantIndex, wantErr := defaultRoute(family)
		if wantErr != nil && wantErr != ErrNotFound {
			t.Errorf("unexpected error from defaultRoute(%d): %v", family, wantErr)
			continue
		}
		iface, index, err := Default(family)
		if err != wantErr {
			t.Errorf("unexpected error from Default(%d): got:%v want:%v", family, err, wantErr)
		}
		if wantErr != nil {
			continue
		}
		if !sameName(iface, wantIface) {
			if family == syscall.AF_INET6 && runtime.GOOS == "windows" {
				// Windows interface naming is a dog's breakfast; on some
				// builders the transport name obtained from getmac is not
				// based on the LUID.
				b, err := run("getmac")
				if err != nil {
					b = []byte(fmt.Sprintf("\nunable to recover getmac information: %v", err))
				}
				t.Logf("unexpected interface for family %d: got:%s want:%s\n%s", family, iface, wantIface, b)
			} else {
				t.Errorf("unexpected interface for family %d: got:%s want:%s", family, iface, wantIface)
			}
		}
		if index != wantIndex {
			t.Errorf("unexpected interface for family %d: got:%d want:%d", family, index, wantIndex)
		}
	}
}

func sameName(got, want string) bool {
	if runtime.GOOS == "windows" {
		// Rely only on the GUID of the device since the device tree
		// may not be consistent (or present).
		idx := strings.Index(want, "_") // Replace this with strings.Cut.
		if idx > -1 {
			want = want[idx+1:]
		}
	}
	return got == want
}

// This wart exists because golangci-lint does not aggregate across
// arches and run is not used in the linux test code.
var _ = run

func run(command string, args ...string) ([]byte, error) {
	cmd := execabs.Command(command, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
