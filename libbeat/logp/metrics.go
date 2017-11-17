package logp

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/elastic/beats/libbeat/monitoring"
)

// logMetrics logs at Info level the integer expvars that have changed in the
// last interval. For each expvar, the delta from the beginning of the interval
// is logged.
func logMetrics(metricsCfg *LoggingMetricsConfig) {
	if metricsCfg.Enabled != nil && *metricsCfg.Enabled == false {
		Info("Metrics logging disabled")
		return
	}
	if metricsCfg.Period == nil {
		metricsCfg.Period = &defaultMetricsPeriod
	}
	Info("Metrics logging every %s", metricsCfg.Period)

	ticker := time.NewTicker(*metricsCfg.Period)

	prevVals := monitoring.MakeFlatSnapshot()
	for range ticker.C {
		snapshot := snapshotMetrics()
		delta := snapshotDelta(prevVals, snapshot)
		prevVals = snapshot

		if len(delta) == 0 {
			Info("No non-zero metrics in the last %s", metricsCfg.Period)
			continue
		}

		metrics := formatMetrics(delta)
		Info("Non-zero metrics in the last %s:%s", metricsCfg.Period, metrics)
	}
}

// LogTotalExpvars logs all registered expvar metrics.
func LogTotalExpvars(cfg *Logging) {
	if cfg.Metrics.Enabled != nil && *cfg.Metrics.Enabled == false {
		return
	}

	zero := monitoring.MakeFlatSnapshot()
	metrics := formatMetrics(snapshotDelta(zero, snapshotMetrics()))
	Info("Total non-zero values: %s", metrics)
	Info("Uptime: %s", time.Since(startTime))
}

func snapshotMetrics() monitoring.FlatSnapshot {
	return monitoring.CollectFlatSnapshot(monitoring.Default, monitoring.Full, true)
}

// List of metrics that are gauges, so that we know for which to
// _not_ subtract the previous value in the output.
// TODO: Replace this with a proper solution that uses the metric
// type from where it is defined. See:
// https://github.com/elastic/beats/issues/5433
var gauges = map[string]bool{
	"libbeat.pipeline.events.active": true,
	"libbeat.pipeline.clients":       true,
	"libbeat.config.module.running":  true,
	"registrar.states.current":       true,
	"filebeat.harvester.running":     true,
	"filebeat.harvester.open_files":  true,
	"beat.memstats.memory_total":     true,
	"beat.memstats.memory_alloc":     true,
	"beat.memstats.gc_next":          true,
}

func snapshotDelta(prev, cur monitoring.FlatSnapshot) map[string]interface{} {
	out := map[string]interface{}{}

	for k, b := range cur.Bools {
		if p, ok := prev.Bools[k]; !ok || p != b {
			out[k] = b
		}
	}

	for k, s := range cur.Strings {
		if p, ok := prev.Strings[k]; !ok || p != s {
			out[k] = s
		}
	}

	for k, i := range cur.Ints {
		if _, found := gauges[k]; found {
			out[k] = i
		} else {
			if p := prev.Ints[k]; p != i {
				out[k] = i - p
			}
		}
	}

	for k, f := range cur.Floats {
		if p := prev.Floats[k]; p != f {
			out[k] = f - p
		}
	}

	return out
}

func formatMetrics(ms map[string]interface{}) string {
	keys := make([]string, 0, len(ms))
	for key := range ms {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteByte(' ')
		buf.WriteString(key)
		buf.WriteString("=")
		buf.WriteString(fmt.Sprintf("%v", ms[key]))
	}
	return buf.String()
}
