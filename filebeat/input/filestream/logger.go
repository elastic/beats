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
	fields := make([]zap.Field, 0, 6)
	fields = append(fields,
		zap.String("operation", event.Op.String()),
		zap.String("source_file", event.SrcID),
	)
	// Log the fingerprint material directly rather than via Key(): this runs on
	// every event (even when the debug line is not emitted), and Key() would
	// hash Raw on the growing-mode hot path. Sum/Raw are already-allocated
	// strings, so this is allocation-free.
	if fp := event.Descriptor.Fingerprint; fp.Complete {
		fields = append(fields, zap.String("fingerprint", fp.Sum))
	} else if fp.Raw != "" {
		fields = append(fields, zap.String("fingerprint", fp.Raw))
	}
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
