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
	"github.com/elastic/elastic-agent-libs/logp"
)

type lazyLog struct {
	log      *logp.Logger
	event    loginp.FSEvent
	enriched bool
}

func (l *lazyLog) Debugf(format string, args ...interface{}) {
	if l.log.IsDebug() {
		l.enrich().Debugf(format, args...)
	}
}

func (l *lazyLog) Warnf(format string, args ...interface{}) {
	l.enrich().Warnf(format, args...)
}

func (l *lazyLog) Errorf(format string, args ...any) {
	l.enrich().Errorf(format, args...)
}

func (l *lazyLog) enrich() *logp.Logger {
	if !l.enriched {
		l.enriched = true
		fields := make([]any, 0, 12)
		fields = append(fields, "operation", l.event.Op.String(), "source_file", l.event.SrcID)

		if l.event.Descriptor.Fingerprint != "" {
			fields = append(fields, "fingerprint", l.event.Descriptor.Fingerprint)
		}
		if l.event.Descriptor.Info != nil {
			if osID := l.event.Descriptor.Info.GetOSState().Identifier(); osID != "" {
				fields = append(fields, "os_id", osID)
			}
		}
		if l.event.NewPath != "" {
			fields = append(fields, "new_path", l.event.NewPath)
		}
		if l.event.OldPath != "" {
			fields = append(fields, "old_path", l.event.OldPath)
		}
		l.log = l.log.With(fields...)
	}
	return l.log
}

func loggerWithEvent(logger *logp.Logger, event loginp.FSEvent) lazyLog {
	return lazyLog{
		log:   logger,
		event: event,
	}
}
