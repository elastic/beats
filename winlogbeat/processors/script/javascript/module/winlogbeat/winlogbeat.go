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

// +build windows

package winlogbeat

import (
	"syscall"
	"unsafe"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// SplitCommandLine splits a string into a list of space separated arguments.
// See Window's CommandLineToArgvW for more details.
func SplitCommandLine(cmd string) []string {
	args, err := commandLineToArgvW(cmd)
	if err != nil {
		panic(err)
	}

	return args
}

func commandLineToArgvW(in string) ([]string, error) {
	ptr, err := syscall.UTF16PtrFromString(in)
	if err != nil {
		return nil, err
	}

	var numArgs int32
	argsWide, err := syscall.CommandLineToArgv(ptr, &numArgs)
	if err != nil {
		return nil, err
	}

	// Free memory allocated for CommandLineToArgvW arguments.
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(argsWide)))

	args := make([]string, numArgs)
	for idx := range args {
		args[idx] = syscall.UTF16ToString(argsWide[idx][:])
	}
	return args, nil
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
