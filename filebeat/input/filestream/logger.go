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
	"github.com/elastic/elastic-agent-libs/logp"
)

func loggerWithEvent(logger *logp.Logger, event loginp.FSEvent, src loginp.Source) *logp.Logger {
	log := logger.With(
		"operation", event.Op.String(),
		"source_name", src.Name(),
	)
	if event.Descriptor.Fingerprint != "" {
		fp := event.Descriptor.Fingerprint
		isGrowingFP := false
		if fs, ok := src.(fileSource); ok {
			isGrowingFP = fs.identifierGenerator == growingFingerprintName
		}
		if isGrowingFP {
			hash := sha256.Sum256([]byte(fp))
			hashedFP := hex.EncodeToString(hash[:])
			log.Debugf("growing fingerprint %s hashed to %s", fp, hashedFP)
			fp = hashedFP
		}
		log = log.With("fingerprint", fp)
	}
	if event.Descriptor.Info != nil {
		osID := event.Descriptor.Info.GetOSState().Identifier()
		if osID != "" {
			log = log.With("os_id", osID)
		}
	}
	if event.NewPath != "" {
		log = log.With("new_path", event.NewPath)
	}
	if event.OldPath != "" {
		log = log.With("old_path", event.OldPath)
	}
	return log
}
