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

// +build amd64,cgo arm64,cgo

package darwin

// #include <unistd.h>
// #include <uuid/uuid.h>
import "C"

import (
	"unsafe"

	"github.com/pkg/errors"
)

// MachineID returns the Hardware UUID also accessible via
// About this Mac -> System Report and as the field
// IOPlatformUUID in the output of "ioreg -d2 -c IOPlatformExpertDevice".
func MachineID() (string, error) {
	return getHostUUID()
}

func getHostUUID() (string, error) {
	var uuidC C.uuid_t
	var id [unsafe.Sizeof(uuidC)]C.uchar
	wait := C.struct_timespec{5, 0} // 5 seconds

	ret, err := C.gethostuuid(&id[0], &wait)
	if ret != 0 {
		if err != nil {
			return "", errors.Wrapf(err, "gethostuuid failed with %v", ret)
		}

		return "", errors.Errorf("gethostuuid failed with %v", ret)
	}

	var uuidStringC C.uuid_string_t
	var uuid [unsafe.Sizeof(uuidStringC)]C.char
	_, err = C.uuid_unparse_upper(&id[0], &uuid[0])
	if err != nil {
		return "", errors.Wrap(err, "uuid_unparse_upper failed")
	}

	return C.GoString(&uuid[0]), nil
}
