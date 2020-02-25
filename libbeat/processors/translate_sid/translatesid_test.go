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

package translate_sid

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/winlogbeat/sys"
)

func TestTranslateSID(t *testing.T) {
	var tests = []struct {
		SID         string
		Account     string
		AccountType sys.SIDType
		Domain      string
		Assert      func(*testing.T, *beat.Event, error)
	}{
		{SID: "S-1-5-7", Domain: "NT AUTHORITY", Account: "ANONYMOUS LOGON"},
		{SID: "S-1-0-0", Account: "NULL SID"},
		{SID: "S-1-1-0", Account: "Everyone"},
		{SID: "S-1-5-32-544", Domain: "BUILTIN", Account: "Administrators", AccountType: sys.SidTypeAlias},
		{SID: "S-1-5-113", Domain: "NT AUTHORITY", Account: "Local Account"},
		{SID: "", Assert: assertInvalidSID},
		{SID: "Not a SID", Assert: assertInvalidSID},
		{SID: "S-1-5-2025429265-500", Assert: assertNoMapping},
	}

	for n, tc := range tests {
		t.Run(fmt.Sprintf("test%d_%s", n, tc.SID), func(t *testing.T) {
			p, err := newFromConfig(config{
				Field:             "sid",
				DomainTarget:      "domain",
				AccountNameTarget: "account",
				AccountTypeTarget: "type",
			})
			if err != nil {
				t.Fatal(err)
			}

			evt := &beat.Event{Fields: map[string]interface{}{
				"sid": tc.SID,
			}}

			evt, err = p.Run(evt)
			if tc.Assert != nil {
				tc.Assert(t, evt, err)
				return
			}
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("%v", evt.Fields.StringToPrint())
			assertEqualIgnoreCase(t, tc.Domain, evt.Fields["domain"])
			assertEqualIgnoreCase(t, tc.Account, evt.Fields["account"])
			if tc.AccountType > 0 {
				assert.Equal(t, tc.AccountType.String(), evt.Fields["type"])
			}
		})
	}
}

func BenchmarkProcessor_Run(b *testing.B) {
	p, err := newFromConfig(config{
		Field:             "sid",
		DomainTarget:      "domain",
		AccountNameTarget: "account",
	})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("builtin", func(b *testing.B) {
		evt := &beat.Event{Fields: map[string]interface{}{
			"sid": "S-1-5-7",
		}}

		for i := 0; i < b.N; i++ {
			_, err = p.Run(evt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("no_mapping", func(b *testing.B) {
		evt := &beat.Event{Fields: map[string]interface{}{
			"sid": "S-1-5-2025429265-500",
		}}

		for i := 0; i < b.N; i++ {
			_, err = p.Run(evt)
			if err != windows.ERROR_NONE_MAPPED {
				b.Fatal(err)
			}
		}
	})
}

func assertEqualIgnoreCase(t *testing.T, expected string, actual interface{}) {
	t.Helper()
	actualStr, ok := actual.(string)
	if !ok {
		assert.Fail(t, "actual value is not a string: %T %#v", actual, actual)
	}
	assert.Equal(t, strings.ToLower(expected), strings.ToLower(actualStr))
}

func assertInvalidSID(t *testing.T, event *beat.Event, err error) {
	if assert.Error(t, err) {
		// The security ID structure is invalid.
		assert.Equal(t, windows.ERROR_INVALID_SID, err)
	}
	assert.Nil(t, event.Fields["domain"])
	assert.Nil(t, event.Fields["account"])
	assert.Nil(t, event.Fields["type"])
}

func assertNoMapping(t *testing.T, event *beat.Event, err error) {
	if assert.Error(t, err) {
		// No mapping between account names and security IDs was done.
		assert.Equal(t, windows.ERROR_NONE_MAPPED, err)
	}
	assert.Nil(t, event.Fields["domain"])
	assert.Nil(t, event.Fields["account"])
	assert.Nil(t, event.Fields["type"])
}
