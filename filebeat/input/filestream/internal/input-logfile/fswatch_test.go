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

package input_logfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameFile(t *testing.T) {
	path := "/var/log/app.log"
	shortFingerprint := "aabb"
	extendedFingerprint := "aabbccdd"
	sha256Fingerprint := "1111111111111111111111111111111111111111111111111111111111111111"
	growingFingerprintHex := "aabbccdd" + "ee00" + // raw-hex of bytes[offset:offset+length], starts with shortFP
		"0000000000000000000000000000000000000000000000000000000000"

	cases := map[string]struct {
		prev    FileDescriptor
		current FileDescriptor
		want    bool
	}{
		"exact FileID match": {
			prev:    FileDescriptor{Filename: path, Fingerprint: sha256Fingerprint},
			current: FileDescriptor{Filename: path, Fingerprint: sha256Fingerprint},
			want:    true,
		},
		"growing-phase prefix match with same filename": {
			prev:    FileDescriptor{Filename: path, Fingerprint: shortFingerprint},
			current: FileDescriptor{Filename: path, Fingerprint: extendedFingerprint},
			want:    true,
		},
		"growing-phase prefix match across rename (path differs)": {
			// Fingerprint is the identity; a renamed file with an extending
			// raw-hex prefix is still the same file.
			prev:    FileDescriptor{Filename: path, Fingerprint: shortFingerprint},
			current: FileDescriptor{Filename: "/var/log/other.log", Fingerprint: extendedFingerprint},
			want:    true,
		},
		"threshold transition: raw-hex prefix of GrowingFingerprint": {
			prev: FileDescriptor{Filename: path, Fingerprint: shortFingerprint},
			current: FileDescriptor{
				Filename:           path,
				Fingerprint:        sha256Fingerprint,
				GrowingFingerprint: growingFingerprintHex,
			},
			want: true,
		},
		"threshold transition: matches even on rename (path differs)": {
			prev: FileDescriptor{Filename: path, Fingerprint: shortFingerprint},
			current: FileDescriptor{
				Filename:           "/var/log/other.log",
				Fingerprint:        sha256Fingerprint,
				GrowingFingerprint: growingFingerprintHex,
			},
			want: true,
		},
		"threshold transition: GrowingFingerprint absent fails": {
			prev: FileDescriptor{Filename: path, Fingerprint: shortFingerprint},
			current: FileDescriptor{
				Filename:    path,
				Fingerprint: sha256Fingerprint,
			},
			want: false,
		},
		"no match: unrelated fingerprints same path": {
			prev:    FileDescriptor{Filename: path, Fingerprint: "ffff"},
			current: FileDescriptor{Filename: path, Fingerprint: "0000"},
			want:    false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := SameFile(&tc.prev, &tc.current)
			assert.Equal(t, tc.want, got, "SameFile result mismatch")
		})
	}
}
