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

type fingerprintIdentifier struct {
	log *logp.Logger
}

func newFingerprintIdentifier(cfg *conf.C) (fileIdentifier, error) {
	return &fingerprintIdentifier{
		log: logp.NewLogger("fingerprint_identifier"),
	}, nil
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
