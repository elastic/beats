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

package winlogbeat

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// SplitCommandLine splits string in a list of space separated arguments.
// Double-quoted arguments are kept together even if it contains spaces.
func SplitCommandLine(cmd string) []string {
	var args []string
	var insideQuotes bool
	var tokenStart = 0

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		switch {
		case c == '"':
			if !insideQuotes {
				tokenStart = i + 1
			} else {
				if i-tokenStart > 0 {
					args = append(args, cmd[tokenStart:i])
				}
				tokenStart = i + 1
			}
			insideQuotes = !insideQuotes
		case c == ' ' && !insideQuotes:
			if i-tokenStart > 0 {
				args = append(args, cmd[tokenStart:i])
			}
			tokenStart = i + 1
		}
	}
	if len(cmd)-tokenStart > 0 {
		args = append(args, cmd[tokenStart:])
	}

	return args
}

// Require registers the winlogbeat module that has utilities specific to
// Winlogbeat like parsing Windows command lines. It can be accessed using:
//
//    // javascript
//    var winlogbeat = require('winlogbeat');
//
func Require(vm *goja.Runtime, module *goja.Object) {
	o := module.Get("exports").(*goja.Object)

	o.Set("splitCommandLine", SplitCommandLine)
}

// Enable adds path to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("winlogbeat", require.Require(runtime, "winlogbeat"))
}

func init() {
	require.RegisterNativeModule("winlogbeat", Require)
}
