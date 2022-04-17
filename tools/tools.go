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

//go:build tools
// +build tools

// This package contains the tool dependencies of the project.

package tools

import (
	_ "github.com/magefile/mage"
	_ "github.com/pierrre/gotestcover"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/tsg/go-daemon"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
	_ "gotest.tools/gotestsum/cmd"

	_ "github.com/mitchellh/gox"
	_ "golang.org/x/lint/golint"

	_ "go.elastic.co/go-licence-detector"

	_ "github.com/menderesk/go-licenser"

	_ "github.com/menderesk/elastic-agent-libs/dev-tools/mage"
)
