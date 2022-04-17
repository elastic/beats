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

package metrics

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
	"github.com/menderesk/beats/v7/libbeat/monitoring/report/log"
	"github.com/menderesk/beats/v7/libbeat/version"
)

var (
	ephemeralID uuid.UUID
	beatMetrics *monitoring.Registry
)

func init() {
	beatMetrics = monitoring.Default.NewRegistry("beat")
	monitoring.NewFunc(beatMetrics, "info", reportInfo, monitoring.Report)

	var err error
	ephemeralID, err = uuid.NewV4()
	if err != nil {
		logp.Err("Error while generating ephemeral ID for Beat")
	}
}

// EphemeralID returns generated EphemeralID
func EphemeralID() uuid.UUID {
	return ephemeralID
}

func reportInfo(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	delta := time.Since(log.StartTime)
	uptime := int64(delta / time.Millisecond)
	monitoring.ReportNamespace(V, "uptime", func() {
		monitoring.ReportInt(V, "ms", uptime)
	})

	monitoring.ReportString(V, "ephemeral_id", ephemeralID.String())
	monitoring.ReportString(V, "version", version.GetDefaultVersion())
}
