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

//+build linux,cgo

package instance

import (
	"fmt"

	"github.com/coreos/go-systemd/sdjournal"

	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	metrics  *monitoring.Registry
	journals map[string]*sdjournal.Journal
)

// SetupJournalMetrics initializes and registers monitoring functions.
func SetupJournalMetrics() {
	metrics = monitoring.Default.NewRegistry("journalbeat")
	journals = make(map[string]*sdjournal.Journal)

	monitoring.NewFunc(metrics, "journals", reportJournalSizes, monitoring.Report)
}

// AddJournalToMonitor adds a new journal which has to be monitored.
func AddJournalToMonitor(path string, journal *sdjournal.Journal) {
	journals[path] = journal
}

// StopMonitoringJournal stops monitoring the journal under the path.
func StopMonitoringJournal(path string) {
	delete(journals, path)
}

func reportJournalSizes(m monitoring.Mode, V monitoring.Visitor) {
	i := 0
	for path, journal := range journals {
		s, err := journal.GetUsage()
		if err != nil {
			continue
		}

		ns := fmt.Sprintf("journal_%d", i)
		monitoring.ReportNamespace(V, ns, func() {
			monitoring.ReportString(V, "path", path)
			monitoring.ReportInt(V, "size_in_bytes", int64(s))
		})
		i++
	}
}
