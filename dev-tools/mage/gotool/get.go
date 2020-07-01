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

package gotool

type goGet func(opts ...ArgOpt) error

// Get runs `go get` and provides optionals for adding command line arguments.
var Get goGet = runGoGet

func runGoGet(opts ...ArgOpt) error {
	args := buildArgs(opts)
	return runVGo("get", args)
}

func (goGet) Download() ArgOpt          { return flagBoolIf("-d", true) }
func (goGet) Update() ArgOpt            { return flagBoolIf("-u", true) }
func (goGet) Package(pkg string) ArgOpt { return posArg(pkg) }
