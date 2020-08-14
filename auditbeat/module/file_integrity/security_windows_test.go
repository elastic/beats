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

package file_integrity

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestGetSecurityInfo(t *testing.T) {
	// Create a temp file that we will use in checking the owner.
	file, err := ioutil.TempFile("", "go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	// Get the file owner.
	var securityID *syscall.SID
	var securityDescriptor *SecurityDescriptor
	if err = GetSecurityInfo(syscall.Handle(file.Fd()), FileObject,
		OwnerSecurityInformation, &securityID, nil, nil, nil, &securityDescriptor); err != nil {
		t.Fatal(err)
	}

	_, err = securityID.String()
	assert.NoError(t, err)
	_, _, _, err = securityID.LookupAccount("")
	assert.NoError(t, err)

	// Freeing the security descriptor releases the memory used by the SID.
	_, err = syscall.LocalFree((syscall.Handle)(unsafe.Pointer(securityDescriptor)))
	if err != nil {
		t.Fatal(err)
	}
}
