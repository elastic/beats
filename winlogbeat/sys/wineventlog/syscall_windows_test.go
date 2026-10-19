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

//go:build windows

package wineventlog

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvtVariantDataBinary verifies that EvtVariant.Data correctly bounds
// the binary slice using Count, rather than reading to the end of the shared
// render buffer. Without the Count bound, subsequent fields in the buffer
// are appended as hex (see https://github.com/elastic/beats/issues/53260).
func TestEvtVariantDataBinary(t *testing.T) {
	// Lay out a buffer where the binary data occupies the first four bytes
	// and unrelated bytes follow immediately after.
	buf := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF}

	v := EvtVariant{
		Count: 4, // only the first four bytes belong to this field
		Type:  EvtVarTypeBinary,
	}
	v.SetValue(uintptr(unsafe.Pointer(&buf[0])))

	got, err := v.Data(buf)
	require.NoError(t, err, "Data() returned unexpected error")
	assert.Equal(t, "DEADBEEF", got, "binary field must be bounded by Count, not extend to end of buffer")
}
