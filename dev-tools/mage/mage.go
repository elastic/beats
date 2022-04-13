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

package mage

import (
	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

func assertUnchanged(path string) error {
	err := sh.Run("git", "diff", "--exit-code", path)
	if err != nil {
		return fmt.Errorf("failed to assert the unchanged file %q: %w", path, err)
	}

	return nil
}

// runWithStdErr runs a command redirecting its stderr to the console instead of discarding it
func runWithStdErr(command string, args ...string) error {
	_, err := sh.Exec(nil, os.Stdout, os.Stderr, command, args...)
	return err
}
