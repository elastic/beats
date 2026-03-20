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

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestMain(m *testing.M) {
	binPath, err := filepath.Abs("../../filebeat.test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve binary path: %s\n", err)
		os.Exit(1)
	}
	packagePath, err := filepath.Abs("../../")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve package path: %s\n", err)
		os.Exit(1)
	}
	if err := integration.BuildSystemTestBinary(binPath, packagePath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build filebeat test binary: %s\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
