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

package aix

/*
#include <sys/utsname.h>
*/
import "C"

import (
	"strconv"

	"github.com/pkg/errors"
)

var oslevel string

func getKernelVersion() (int, int, error) {
	name := C.struct_utsname{}
	if _, err := C.uname(&name); err != nil {
		return 0, 0, errors.Wrap(err, "kernel version: uname")
	}

	version, err := strconv.Atoi(C.GoString(&name.version[0]))
	if err != nil {
		return 0, 0, errors.Wrap(err, "parsing kernel version")
	}

	release, err := strconv.Atoi(C.GoString(&name.release[0]))
	if err != nil {
		return 0, 0, errors.Wrap(err, "parsing kernel release")
	}
	return version, release, nil
}

// KernelVersion returns the version of AIX kernel
func KernelVersion() (string, error) {
	major, minor, err := getKernelVersion()
	if err != nil {
		return "", err
	}
	return strconv.Itoa(major) + "." + strconv.Itoa(minor), nil
}
