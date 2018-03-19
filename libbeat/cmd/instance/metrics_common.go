package instance

import (
	"time"

	"github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/report/log"
)

var (
	ephemeralID uuid.UUID
	beatMetrics *monitoring.Registry
)

func init() {
	beatMetrics = monitoring.Default.NewRegistry("beat")
	monitoring.NewFunc(beatMetrics, "info", reportInfo, monitoring.Report)

	ephemeralID = uuid.NewV4()
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
}
