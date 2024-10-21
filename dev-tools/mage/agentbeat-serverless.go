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
	"log"
	"os"
)

// TestBeatServerless todo description
func TestBeatServerless(beat string) {
	if beat == "" {
		log.Fatal("Beat is not defined")
	}

	if os.Getenv("AGENT_BUILD_DIR") == "" {
		log.Fatal("AGENT_BUILD_DIR is not defined")
	}

	setStackProvisioner()
	setTestBinaryName(beat)

}

func setStackProvisioner() {
	stackProvisioner := os.Getenv("STACK_PROVISIONER")
	if stackProvisioner == "" {
		if err := os.Setenv("STACK_PROVISIONER", "serverless"); err != nil {
			log.Fatal("error setting serverless stack var: %w", err)
		}
	} else if stackProvisioner == "stateful" {
		fmt.Println("--- Warning: running TestBeatServerless as stateful")
	}
}

func setTestBinaryName(beat string) {
	if err := os.Setenv("TEST_BINARY_NAME", beat); err != nil {
		log.Fatal("error setting binary name: %w", err)
	}
}
