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

	"github.com/magefile/mage/mg"

	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
)

func GenerateNotice(overrides, rules, noticeTemplate string) error {
	mg.Deps(InstallGoNoticeGen, Deps.CheckModuleTidy)

	err := gotool.Mod.Download(gotool.Download.All())
	if err != nil {
		return fmt.Errorf("error while downloading dependencies: %w", err)
	}

	// Ensure the go.mod file is left unchanged after go mod download all runs.
	// go mod download will modify go.sum in a way that conflicts with go mod tidy.
	// https://github.com/golang/go/issues/43994#issuecomment-770053099
	defer gotool.Mod.Tidy() //nolint:errcheck // No value in handling this error.

	out, _ := gotool.ListDepsForNotice()
	depsFile, _ := os.CreateTemp("", "depsout")
	defer os.Remove(depsFile.Name())
	_, _ = depsFile.Write([]byte(out))
	depsFile.Close()

	generator := gotool.NoticeGenerator
	return generator(
		generator.Dependencies(depsFile.Name()),
		generator.IncludeIndirect(),
		generator.Overrides(overrides),
		generator.Rules(rules),
		generator.NoticeTemplate(noticeTemplate),
		generator.NoticeOutput("NOTICE.txt"),
	)
}
