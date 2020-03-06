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

// Mod is the command go mod.
var Mod = goMod{
	Init:   modCommand{"init"}.run,
	Tidy:   modCommand{"tidy"}.run,
	Verify: modCommand{"verify"}.run,
	Vendor: modCommand{"vendor"}.run,
}

type modCommand struct {
	method string
}

func (cmd modCommand) run(opts ...ArgOpt) error {
	o := make([]ArgOpt, len(opts)+1)
	o[0] = posArg(cmd.method)
	for i, opt := range opts {
		o[i+1] = opt
	}
	args := buildArgs(o)
	return runVGo("mod", args)
}

type goMod struct {
	Init   modInit
	Tidy   modTidy
	Verify modVerify
	Vendor modVendor
}

// modInit initializes a new go module in folder.
type modInit func(opts ...ArgOpt) error

// modTidy cleans the go.mod file
type modTidy func(opts ...ArgOpt) error

// modVerify check that deps have the expected content.
type modVerify func(opts ...ArgOpt) error

// modVendor downloads and copies dependencies under the folder vendor.
type modVendor func(opts ...ArgOpt) error
