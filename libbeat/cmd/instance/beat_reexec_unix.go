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

//go:build !windows

package instance

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func (b *Beat) doReexec() error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}

	binary := filepath.Join(pwd, os.Args[0])
	if err := unix.Exec(binary, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("could not exec '%s', err: %w", binary, err)
	}

	return nil
}
