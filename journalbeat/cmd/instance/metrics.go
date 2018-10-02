package instance

import (
	"fmt"

	"github.com/coreos/go-systemd/sdjournal"

	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	metrics  *monitoring.Registry
	journals map[string]sdjournal.Journal
)

func SetupJournalMetrics() {
	metrics = monitoring.Default.NewRegistry("journalbeat")
	journals = make(map[string]sdjournal.Journal)

	monitoring.NewFunc(metrics, "journals", reportJournalSizes, monitoring.Report)
}

func AddJournalToMonitor(path string, journal sdjournal.Journal) {
	journals[path] = journal
}

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
