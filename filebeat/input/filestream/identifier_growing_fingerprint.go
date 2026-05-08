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
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// growingFingerprintIdentifier identifies files by their raw content fingerprint.
// Unlike the hash-based fingerprint identifier, this stores the actual bytes
// (hex-encoded) of the file header and the fingerprint can grow as the file grows.
// This allows tracking files of any size immediately, without waiting for them
// to reach a minimum size threshold.
type growingFingerprintIdentifier struct{}

func newGrowingFingerprintIdentifier(_ *conf.C, _ *logp.Logger) (fileIdentifier, error) {
	return &growingFingerprintIdentifier{}, nil
}

func (i *growingFingerprintIdentifier) GetSource(e loginp.FSEvent) fileSource {
	return fileSource{
		desc:                e.Descriptor,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		archived:            e.Op == loginp.OpArchived,
		fileID:              growingFingerprintName + identitySep + e.Descriptor.Fingerprint,
		identifierGenerator: growingFingerprintName,
	}
}

func (i *growingFingerprintIdentifier) Name() string {
	return growingFingerprintName
}

func (i *growingFingerprintIdentifier) Supports(f identifierFeature) bool {
	switch f {
	case trackRename:
		return true
	default:
		return false
	}
}
