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
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameFile(t *testing.T) {
	path := "/var/log/app.log"
	shortRaw := "aabb"
	extendedRaw := "aabbccdd"
	sha256Fingerprint := "1111111111111111111111111111111111111111111111111111111111111111"
	// raw-hex of bytes[offset:offset+length] on the threshold-crossing scan;
	// starts with shortRaw so the growing predecessor is a strict prefix.
	fullRaw := "aabbccdd" + "ee00" +
		"0000000000000000000000000000000000000000000000000000000000"

	growing := func(raw string) FingerprintID { return FingerprintID{Raw: raw} }
	complete := func(sum, raw string) FingerprintID {
		return FingerprintID{Complete: true, Sum: sum, Raw: raw}
	}

	cases := map[string]struct {
		prev    FileDescriptor
		current FileDescriptor
		want    bool
	}{
		"exact FileID match": {
			prev:    FileDescriptor{Filename: path, Fingerprint: complete(sha256Fingerprint, "")},
			current: FileDescriptor{Filename: path, Fingerprint: complete(sha256Fingerprint, "")},
			want:    true,
		},
		"growing-phase prefix match with same filename": {
			prev:    FileDescriptor{Filename: path, Fingerprint: growing(shortRaw)},
			current: FileDescriptor{Filename: path, Fingerprint: growing(extendedRaw)},
			want:    true,
		},
		"growing-phase prefix match across rename (path differs)": {
			// The fingerprint is the identity; a renamed file with an extending
			// raw-hex prefix is still the same file.
			prev:    FileDescriptor{Filename: path, Fingerprint: growing(shortRaw)},
			current: FileDescriptor{Filename: "/var/log/other.log", Fingerprint: growing(extendedRaw)},
			want:    true,
		},
		"threshold transition: prev raw-hex is a prefix of completed Raw": {
			prev:    FileDescriptor{Filename: path, Fingerprint: growing(shortRaw)},
			current: FileDescriptor{Filename: path, Fingerprint: complete(sha256Fingerprint, fullRaw)},
			want:    true,
		},
		"threshold transition: matches even on rename (path differs)": {
			prev:    FileDescriptor{Filename: path, Fingerprint: growing(shortRaw)},
			current: FileDescriptor{Filename: "/var/log/other.log", Fingerprint: complete(sha256Fingerprint, fullRaw)},
			want:    true,
		},
		"threshold transition: completed Raw dropped fails": {
			// A completed descriptor whose Raw was trimmed (retained state, or
			// static mode) cannot bridge a growing predecessor.
			prev:    FileDescriptor{Filename: path, Fingerprint: growing(shortRaw)},
			current: FileDescriptor{Filename: path, Fingerprint: complete(sha256Fingerprint, "")},
			want:    false,
		},
		"no match: unrelated fingerprints same path": {
			prev:    FileDescriptor{Filename: path, Fingerprint: growing("ffff")},
			current: FileDescriptor{Filename: path, Fingerprint: growing("0000")},
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

func TestFingerprintIDKey(t *testing.T) {
	rawHash := func(raw string) string {
		sum := sha256.Sum256([]byte(raw))
		return hex.EncodeToString(sum[:])
	}

	cases := map[string]struct {
		id   FingerprintID
		want string
	}{
		"empty fingerprint yields empty key": {
			id:   FingerprintID{},
			want: "",
		},
		"completed fingerprint keys on its SHA-256 sum": {
			id:   FingerprintID{Complete: true, Sum: "deadbeef", Raw: "aabbccdd"},
			want: "deadbeef",
		},
		"growing fingerprint keys on a bounded hash of Raw": {
			id:   FingerprintID{Raw: "aabb"},
			want: rawHash("aabb"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.id.Key(), "Key() mismatch")
		})
	}
}

func TestFingerprintIDContinues(t *testing.T) {
	cases := map[string]struct {
		prev    FingerprintID
		current FingerprintID
		want    bool
	}{
		"empty prev never continues": {
			prev:    FingerprintID{},
			current: FingerprintID{Raw: "aabb"},
			want:    false,
		},
		"prefix growth continues": {
			prev:    FingerprintID{Raw: "aabb"},
			current: FingerprintID{Raw: "aabbccdd"},
			want:    true,
		},
		"crossing continues via completed Raw": {
			prev:    FingerprintID{Raw: "aabb"},
			current: FingerprintID{Complete: true, Sum: "1111", Raw: "aabbccdd"},
			want:    true,
		},
		"completed current with dropped Raw does not continue": {
			prev:    FingerprintID{Raw: "aabb"},
			current: FingerprintID{Complete: true, Sum: "1111"},
			want:    false,
		},
		"unrelated does not continue": {
			prev:    FingerprintID{Raw: "ffff"},
			current: FingerprintID{Raw: "0000"},
			want:    false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.prev.Continues(tc.current), "Continues() mismatch")
		})
	}
}
