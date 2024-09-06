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

package mongodb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSetFlagBits(t *testing.T) {
	tests := []struct {
		name     string
		flagBits int32
		wantMsg  mongodbMessage
	}{
		{
			name:     "none",
			flagBits: 0b0000,
			wantMsg:  mongodbMessage{},
		},
		{
			name:     "checksumpresent",
			flagBits: 0b0001,
			wantMsg:  mongodbMessage{checkSumPresent: true},
		},
		{
			name:     "moreToCome",
			flagBits: 0b00010,
			wantMsg:  mongodbMessage{moreToCome: true},
		},
		{
			name:     "checksumpresent_moreToCome",
			flagBits: 0b00011,
			wantMsg:  mongodbMessage{checkSumPresent: true, moreToCome: true},
		},
		{
			name:     "exhaustallowed",
			flagBits: 0x10000,
			wantMsg:  mongodbMessage{exhaustAllowed: true},
		},
		{
			name:     "checksumpresent_exhaustallowed",
			flagBits: 0x10001,
			wantMsg:  mongodbMessage{checkSumPresent: true, exhaustAllowed: true},
		},
		{
			name:     "checksumpresent_moreToCome_exhaustallowed",
			flagBits: 0x10003,
			wantMsg:  mongodbMessage{checkSumPresent: true, moreToCome: true, exhaustAllowed: true},
		},
	}

	flagBitsComparer := cmp.Comparer(func(m1, m2 mongodbMessage) bool {
		return m1.checkSumPresent == m2.checkSumPresent &&
			m1.moreToCome == m2.moreToCome &&
			m1.exhaustAllowed == m2.exhaustAllowed
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMsg mongodbMessage
			gotMsg.SetFlagBits(tc.flagBits)

			diff := cmp.Diff(tc.wantMsg, gotMsg, flagBitsComparer)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
