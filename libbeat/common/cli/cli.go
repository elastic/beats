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
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func exitOnPanic() {
	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "panic: %s\n", r)
		debug.PrintStack()
		os.Exit(1)
	}
}

// RunWith wrap cli function with an error handler instead of having the code exit early.
func RunWith(
	fn func(cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		defer exitOnPanic()
		if err := fn(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}

// GetEnvOr return the value of the environment variable if the value is set, if its not set it will
// return the default value.
//
// Note: if the value is set but it is an empty string we will return the empty string.
func GetEnvOr(name, def string) string {
	if env, ok := os.LookupEnv(name); ok {
		return env
	}
	return def
}
