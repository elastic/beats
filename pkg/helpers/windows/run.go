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

//go:build ignore
// +build ignore

// This is a simple wrapper to set the GOARCH environment variable, execute the
// given command, and output the stdout to a file. It's the unix equivalent of
// GOARCH=arch command > output.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	goarch = flag.String("goarch", "", "GOARCH value")
	output = flag.String("output", "", "output file")
	cmd    = flag.String("cmd", "", "command to run")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Run command.
	parts := strings.Fields(*cmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()

	if *goarch != "" {
		cmd.Env = append(cmd.Env, "GOARCH="+*goarch)
	}

	if *output != "" {
		outputBytes, err := cmd.Output()
		if err != nil {
			return err
		}

		// Write output.
		f, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(outputBytes)
		return err
	}

	cmd.Stdout = cmd.Stdout
	return cmd.Run()
}
