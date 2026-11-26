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

package filestream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

func TestFingerprintIdentifierGetSourceRaw(t *testing.T) {
	const (
		raw = "00112233445566778899aabbccddeeff"
		sum = "completesum"
	)

	tcs := []struct {
		name        string
		fingerprint loginp.FingerprintID
		wantSrcRaw  string
		wantFileID  string
	}{
		{
			name:        "completed fingerprint drops Raw from the source",
			fingerprint: loginp.FingerprintID{Raw: raw, Sum: sum},
			wantSrcRaw:  "",
			wantFileID:  fingerprintName + identitySep + sum,
		},
		{
			name:        "completed fingerprint without Raw",
			fingerprint: loginp.FingerprintID{Sum: sum},
			wantSrcRaw:  "",
			wantFileID:  fingerprintName + identitySep + sum,
		},
		{
			name:        "growing fingerprint keeps Raw in the source",
			fingerprint: loginp.FingerprintID{Raw: raw},
			wantSrcRaw:  raw,
			wantFileID:  fingerprintName + identitySep + loginp.HashRawFingerprint(raw),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			identifier, err := newFingerprintIdentifier(nil, nil)
			require.NoError(t, err, "newFingerprintIdentifier failed")

			event := loginp.FSEvent{
				NewPath:    "/var/log/a.log",
				Op:         loginp.OpCreate,
				Descriptor: loginp.FileDescriptor{Fingerprint: tc.fingerprint},
			}

			src := identifier.GetSource(event)

			assert.Equal(t, tc.wantSrcRaw, src.desc.Fingerprint.Raw,
				"unexpected Raw retained by the source descriptor")
			assert.Equal(t, tc.fingerprint.Sum, src.desc.Fingerprint.Sum,
				"source descriptor Sum must be preserved")
			assert.Equal(t, tc.fingerprint.Complete(), src.desc.Fingerprint.Complete(),
				"source descriptor completeness must be preserved")
			assert.Equal(t, tc.fingerprint.Key(), src.desc.Fingerprint.Key(),
				"source descriptor identity key must match the event's")
			assert.Equal(t, tc.wantFileID, src.fileID,
				"unexpected source fileID")
			assert.Equal(t, tc.wantFileID, src.Name(),
				"unexpected source Name()")
		})
	}
}
