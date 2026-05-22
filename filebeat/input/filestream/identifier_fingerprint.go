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
	"fmt"

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

func newFingerprintIdentifier(cfg *conf.C, _ *logp.Logger) (fileIdentifier, error) {
	// Parse the sub-config to validate the fields users may set on the
	// fingerprint identity (e.g. growing). The values themselves are read
	// later in normalizeConfig and propagated to the scanner config; the
	// identifier itself does not currently need them at runtime, but
	// unpacking here surfaces config errors at the right point.
	fpCfg := defaultFingerprintIdentityConfig()
	if cfg != nil {
		if err := cfg.Unpack(&fpCfg); err != nil {
			return nil, fmt.Errorf("invalid file_identity.fingerprint config: %w", err)
		}
	}
	return &fingerprintIdentifier{}, nil
}

func (i *fingerprintIdentifier) GetSource(e loginp.FSEvent) fileSource {
	return fileSource{
		desc:                e.Descriptor,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		archived:            e.Op == loginp.OpArchived,
		fileID:              fingerprintName + identitySep + e.Descriptor.Fingerprint,
		identifierGenerator: fingerprintName,
	}
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
