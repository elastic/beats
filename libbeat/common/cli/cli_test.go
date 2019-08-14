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

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func runCli(testName string) (*bytes.Buffer, error) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "TEST_RUNWITH=1")
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	err := cmd.Run()
	return stderr, err
}

// Example taken from slides from Andrew Gerrand
// https://talks.golang.org/2014/testing.slide#23
func TestExitWithError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("Something bad")
			})(cmd, args)
		}()
		return
	}

	stderr, err := runCli("TestExitWithError")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "exit status 1")
	}
	assert.Equal(t, "Something bad\n", stderr.String())
}

func TestExitWithoutError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				return nil
			})(cmd, args)
		}()
		return
	}

	stderr, err := runCli("TestExitWithoutError")
	assert.NoError(t, err)
	assert.Equal(t, "", stderr.String())
}

func TestExitWithPanic(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				panic("something really bad happened")
			})(cmd, args)
		}()
		return
	}

	stderr, err := runCli("TestExitWithPanic")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "exit status 1")
	}
	assert.Contains(t, stderr.String(), "something really bad happened")
}
