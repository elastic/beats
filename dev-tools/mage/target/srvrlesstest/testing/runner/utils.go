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

package runner

import (
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/core/process"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// WorkDir returns the current absolute working directory.
func WorkDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get work directory: %w", err)
	}
	wd, err = filepath.Abs(wd)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path to work directory: %w", err)
	}
	return wd, nil
}

func AttachOut(w io.Writer) process.CmdOption {
	return func(c *exec.Cmd) error {
		c.Stdout = w
		return nil
	}
}

func AttachErr(w io.Writer) process.CmdOption {
	return func(c *exec.Cmd) error {
		c.Stderr = w
		return nil
	}
}
