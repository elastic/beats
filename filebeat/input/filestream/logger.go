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
	"go.uber.org/zap"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp"
)

// loggerWithEvent returns a logger enriched with FS-event fields. Enrichment
// is deferred via [logp.Logger.WithLazy], so the underlying core only pays
// for the fields if a message is actually emitted.
func loggerWithEvent(logger *logp.Logger, event loginp.FSEvent) *logp.Logger {
	// The file is identified by path (new_path/old_path below). The source name
	// (event.SrcID) and the fingerprint are intentionally NOT logged: SrcID
	// embeds the fingerprint, and the fingerprint may contain raw file bytes.
	fields := make([]zap.Field, 0, 4)
	fields = append(fields,
		zap.String("operation", event.Op.String()),
	)
	if info := event.Descriptor.Info; info != nil {
		if osID := info.GetOSState().Identifier(); osID != "" {
			fields = append(fields, zap.String("os_id", osID))
		}
	}
	if event.NewPath != "" {
		fields = append(fields, zap.String("new_path", event.NewPath))
	}
	if event.OldPath != "" {
		fields = append(fields, zap.String("old_path", event.OldPath))
	}
	return logger.WithLazy(fields...)
}
