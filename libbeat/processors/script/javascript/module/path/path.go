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

package path

import (
	"path"
	"path/filepath"
	"runtime"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// Require registers the path module that provides utilities for working with
// file and directory paths. It can be accessed using:
//
//    // javascript
//    var path = require('path');
//
func Require(vm *goja.Runtime, module *goja.Object) {
	setPosix := func(o *goja.Object) *goja.Object {
		o.Set("basename", path.Base)
		o.Set("dirname", path.Dir)
		o.Set("extname", path.Ext)
		o.Set("isAbsolute", path.IsAbs)
		o.Set("normalize", path.Clean)
		o.Set("sep", '/')
		return o
	}

	setWin32 := func(o *goja.Object) *goja.Object {
		o.Set("basename", win32.Base)
		o.Set("dirname", win32.Dir)
		o.Set("extname", filepath.Ext)
		o.Set("isAbsolute", win32.IsAbs)
		o.Set("normalize", win32.Clean)
		o.Set("sep", win32Separator)
		return o
	}

	o := module.Get("exports").(*goja.Object)
	o.Set("posix", setPosix(vm.NewObject()))
	o.Set("win32", setWin32(vm.NewObject()))

	if runtime.GOOS == "windows" {
		setWin32(o)
	} else {
		setPosix(o)
	}
}

// Enable adds path to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("path", require.Require(runtime, "path"))
}

func init() {
	require.RegisterNativeModule("path", Require)
}
