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
	"crypto/sha256"
	"encoding/hex"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// fingerprintIdentityConfig holds the user-facing configuration for the
// fingerprint file identity. The fields are propagated to the scanner's
// fingerprint config via normalizeConfig before the prospector is created.
type fingerprintIdentityConfig struct {
	// Growing opts into the Enhanced Fingerprint behavior: files smaller
	// than the configured fingerprint size (prospector.scanner.fingerprint.
	// offset + prospector.scanner.fingerprint.length) are tracked using the
	// raw bytes available so far, instead of being skipped as too small.
	// Once such a file grows past the threshold it is automatically rekeyed
	// to the same SHA-256 hex the static fingerprint produces, so existing
	// static-fingerprint state is reused with no data duplication.
	//
	// Default: true (9.5+). Set to `false` to fall back to the legacy
	// static-fingerprint behavior where small files are dropped until
	// they reach offset+length.
	Growing bool `config:"growing"`
}

// defaultFingerprintIdentityConfig returns the default configuration for the
// fingerprint file identity. Growing defaults to `true` on 9.5+; and on
// 8.19.x it defaults to `false`.
func defaultFingerprintIdentityConfig() fingerprintIdentityConfig {
	return fingerprintIdentityConfig{
		Growing: true,
	}
}

type fingerprintIdentifier struct {
}

// newFingerprintIdentifier constructs the fingerprint file identifier. The
// `file_identity.fingerprint` sub-config (e.g. `growing`) is consumed and
// validated later by normalizeConfig; the identifier itself does not
// currently need those values at runtime.
func newFingerprintIdentifier(_ *conf.C, _ *logp.Logger) (fileIdentifier, error) {
	return &fingerprintIdentifier{}, nil
}

func (i *fingerprintIdentifier) GetSource(e loginp.FSEvent) fileSource {
	return fileSource{
		desc:                e.Descriptor,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		archived:            e.Op == loginp.OpArchived,
		fileID:              formatIdentity(fingerprintName, boundFingerprintKey(e.Descriptor)),
		identifierGenerator: fingerprintName,
	}
}

// boundFingerprintKey returns the fixed-size component used in the registry key.
//
// A final SHA-256 fingerprint (FingerprintGrowing == false) is already a
// 64-char hex string and is used as-is, preserving compatibility with static
// fingerprint state. A growing fingerprint is the raw hex of the file header
// and grows up to 2*length characters; hashing it keeps the registry key (and
// therefore every frequent cursor write to the memlog WAL and every checkpoint)
// bounded. The raw growing fingerprint is persisted separately in the entry
// value (fileMeta.Fingerprint) so prefix matching still works after a restart.
func boundFingerprintKey(d loginp.FileDescriptor) string {
	if !d.FingerprintGrowing || d.Fingerprint == "" {
		return d.Fingerprint
	}
	sum := sha256.Sum256([]byte(d.Fingerprint))
	return hex.EncodeToString(sum[:])
}

func (i *fingerprintIdentifier) Name() string {
	return fingerprintName
}

func (i *fingerprintIdentifier) Supports(f identifierFeature) bool {
	switch f {
	case trackRename:
		return true
	default:
	}
	return false
}
