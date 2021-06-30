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
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func loggerWithEvent(logger *logp.Logger, event loginp.FSEvent, src loginp.Source) *logp.Logger {
	log := logger.With(
		"operation", event.Op,
		"source_name", src.Name(),
	)
	if event.Info != nil && event.Info.Sys() != nil {
		log = log.With("os_id", file.GetOSState(event.Info))
	}
	if event.NewPath != "" {
		log = log.With("new_path", event.NewPath)
	}
	if event.OldPath != "" {
		log = log.With("old_path", event.OldPath)
	}
	return log
}
