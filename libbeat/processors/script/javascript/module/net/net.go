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

package net

import (
	"net"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// Require registers the net module that provides utilities for working with IP
// addresses. It can be accessed using:
//
//    // javascript
//    var net = require('net');
//
func Require(vm *goja.Runtime, module *goja.Object) {
	o := module.Get("exports").(*goja.Object)
	o.Set("isIP", isIP)
	o.Set("isIPv4", isIPv4)
	o.Set("isIPv6", isIPv6)
}

func isIP(input string) int32 {
	ip := net.ParseIP(input)
	if ip == nil {
		return 0
	}

	if ip.To4() != nil {
		return 4
	}

	return 6
}

func isIPv4(input string) bool {
	return 4 == isIP(input)
}

func isIPv6(input string) bool {
	return 6 == isIP(input)
}

// Enable adds net to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("net", require.Require(runtime, "net"))
}

func init() {
	require.RegisterNativeModule("net", Require)
}
