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

package generate

import (
	"fmt"
	"os"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"
	genmod "github.com/elastic/beats/v8/filebeat/generator/module"
)

// Module creates a new Filebeat module.
// Use MODULE=module to specify the name of the new module
func Module() error {
	targetModule := os.Getenv("MODULE")
	if targetModule == "" {
		return fmt.Errorf("you must specify the module: MODULE=name mage generate:module")
	}

	ossDir := devtools.OSSBeatDir()
	xPackDir := devtools.XPackBeatDir()

	switch devtools.CWD() {
	case ossDir:
		return genmod.Generate(targetModule, ossDir, ossDir)
	case xPackDir:
		return genmod.Generate(targetModule, xPackDir, ossDir)
	default:
		return fmt.Errorf("you must be in a filebeat directory")
	}
}
