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
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// maxInsertStrings is the maximum number of parameters supported in a
	// Windows event message.
	maxInsertStrings = 99

	leftTemplateDelim  = "[{{"
	rightTemplateDelim = "}}]"
)

// templateInserts contains EvtVariant values that can be used to substitute
// Go text/template expressions into a Windows event message.
var templateInserts = newTemplateStringInserts()

// stringsInserts holds EvtVariant values with type EvtVarTypeString.
type stringInserts struct {
	// insertStrings are slices holding the strings in the EvtVariant (this must
	// keep a reference to these to prevent GC of the strings as there is
	// an unsafe reference to them in the evtVariants).
	insertStrings [maxInsertStrings][]uint16
	evtVariants   [maxInsertStrings]EvtVariant
}

// Slice returns a slice of the full EvtVariant array.
func (si *stringInserts) Slice() []EvtVariant {
	return si.evtVariants[:]
}

// clear clears the pointers (and unsafe pointers) so that the memory can be
// garbage collected.
func (si *stringInserts) clear() {
	for i := 0; i < len(si.evtVariants); i++ {
		si.evtVariants[i] = EvtVariant{}
		si.insertStrings[i] = nil
	}
}

// newTemplateStringInserts returns a stringInserts where each value is a
// Go text/template expression that references an event data parameter.
func newTemplateStringInserts() *stringInserts {
	si := &stringInserts{}

	for i := 0; i < len(si.evtVariants); i++ {
		// Use i+1 to keep our inserts numbered the same as Window's inserts.
		templateParam := leftTemplateDelim + `eventParam $ ` + strconv.Itoa(i+1) + rightTemplateDelim
		strSlice, err := windows.UTF16FromString(templateParam)
		if err != nil {
			// This will never happen.
			panic(err)
		}

		si.insertStrings[i] = strSlice
		si.evtVariants[i] = EvtVariant{
			Count: uint32(len(strSlice)),
			Type:  EvtVarTypeString,
		}
		si.evtVariants[i].SetValue(uintptr(unsafe.Pointer(&strSlice[0])))
		si.evtVariants[i].Type = EvtVarTypeString
	}

	return si
}
