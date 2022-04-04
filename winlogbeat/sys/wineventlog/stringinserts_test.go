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
// +build windows

package wineventlog

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestStringInserts(t *testing.T) {
	assert.NotNil(t, templateInserts)

	si := newTemplateStringInserts()
	defer si.clear()

	// "The value of n can be a number between 1 and 99."
	// https://docs.microsoft.com/en-us/windows/win32/eventlog/message-text-files
	assert.Contains(t, windows.UTF16ToString(si.insertStrings[0]), " 1}")
	assert.Contains(t, windows.UTF16ToString(si.insertStrings[maxInsertStrings-1]), " 99}")

	for i, evtVariant := range si.evtVariants {
		assert.EqualValues(t, uintptr(unsafe.Pointer(&si.insertStrings[i][0])), evtVariant.ValueAsUintPtr())
		assert.Len(t, si.insertStrings[i], int(evtVariant.Count))
		assert.Equal(t, evtVariant.Type, EvtVarTypeString)
	}
}
